package com.shell.miner.data.repository

import com.shell.miner.domain.MiningShare
import com.shell.miner.domain.MiningWork
import com.shell.miner.domain.PoolClient
import kotlinx.coroutines.*
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import timber.log.Timber
import java.io.*
import java.net.Socket
import java.util.concurrent.atomic.AtomicBoolean
import java.util.concurrent.atomic.AtomicLong
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class PoolClientImpl @Inject constructor() : PoolClient {

    private var socket: Socket? = null
    private var writer: PrintWriter? = null
    private var reader: BufferedReader? = null
    private val isConnected = AtomicBoolean(false)
    private val messageId = AtomicLong(1)
    private var connectionJob: Job? = null
    private var heartbeatJob: Job? = null

    private val json = Json {
        ignoreUnknownKeys = true
        encodeDefaults = true
    }

    override suspend fun connect(poolUrl: String): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            if (isConnected.get()) {
                return@withContext Result.success(Unit)
            }

            // Parse pool URL - format: stratum+tcp://host:port
            val cleanUrl = poolUrl.removePrefix("stratum+tcp://")
            val parts = cleanUrl.split(":")
            if (parts.size != 2) {
                return@withContext Result.failure(IllegalArgumentException("Invalid pool URL format"))
            }

            val host = parts[0]
            val port = parts[1].toInt()

            // Create socket connection
            socket = Socket(host, port).apply {
                soTimeout = 30000 // 30 second timeout
                keepAlive = true
            }

            writer = PrintWriter(socket!!.getOutputStream(), true)
            reader = BufferedReader(InputStreamReader(socket!!.getInputStream()))

            // Perform mining subscription
            val subscribeResult = performSubscription()
            if (!subscribeResult) {
                disconnect()
                return@withContext Result.failure(Exception("Failed to subscribe to pool"))
            }

            // Authorize mining
            val authorizeResult = performAuthorization()
            if (!authorizeResult) {
                disconnect()
                return@withContext Result.failure(Exception("Failed to authorize with pool"))
            }

            isConnected.set(true)

            // Start connection monitoring
            startConnectionMonitoring()

            Timber.i("Successfully connected to pool: $poolUrl")
            Result.success(Unit)
        } catch (e: Exception) {
            Timber.e(e, "Failed to connect to pool: $poolUrl")
            disconnect()
            Result.failure(e)
        }
    }

    override suspend fun disconnect(): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            isConnected.set(false)
            
            // Cancel background jobs
            connectionJob?.cancel()
            heartbeatJob?.cancel()
            
            // Close connections
            writer?.close()
            reader?.close()
            socket?.close()

            writer = null
            reader = null
            socket = null

            Timber.i("Disconnected from pool")
            Result.success(Unit)
        } catch (e: Exception) {
            Timber.e(e, "Error during pool disconnect")
            Result.failure(e)
        }
    }

    override suspend fun getWork(): Result<MiningWork> = withContext(Dispatchers.IO) {
        try {
            if (!isConnected.get()) {
                return@withContext Result.failure(Exception("Not connected to pool"))
            }

            // Request work using mobile-optimized getwork method
            val request = StratumRequest(
                id = messageId.getAndIncrement(),
                method = "mining.get_mobile_work",
                params = listOf("mobile_client_v1.0")
            )

            val response = sendRequest(request)
            if (response?.error != null) {
                return@withContext Result.failure(Exception("Pool error: ${response.error}"))
            }

            val workData = response?.result as? Map<*, *>
                ?: return@withContext Result.failure(Exception("Invalid work response"))

            val work = MiningWork(
                jobId = workData["job_id"] as? String ?: "",
                blockHeader = workData["block_header"] as? String ?: "",
                target = workData["target"] as? String ?: "",
                difficulty = (workData["difficulty"] as? Number)?.toDouble() ?: 1.0,
                timestamp = System.currentTimeMillis()
            )

            Timber.d("Received work: jobId=${work.jobId}, difficulty=${work.difficulty}")
            Result.success(work)
        } catch (e: Exception) {
            Timber.e(e, "Failed to get work from pool")
            Result.failure(e)
        }
    }

    override suspend fun submitWork(share: MiningShare): Result<Boolean> = withContext(Dispatchers.IO) {
        try {
            if (!isConnected.get()) {
                return@withContext Result.failure(Exception("Not connected to pool"))
            }

            // Submit share with mobile-specific thermal proof
            val params = mutableListOf<Any>(
                "mobile_miner",
                share.nonce,
                share.hash,
                share.difficulty,
                share.timestamp
            )

            // Add thermal proof if available (mobile-specific)
            share.thermalProof?.let { params.add(it) }

            val request = StratumRequest(
                id = messageId.getAndIncrement(),
                method = "mining.submit_mobile_share",
                params = params
            )

            val response = sendRequest(request)
            if (response?.error != null) {
                Timber.w("Share rejected: ${response.error}")
                return@withContext Result.success(false)
            }

            val accepted = response?.result as? Boolean ?: false
            if (accepted) {
                Timber.d("Share accepted: nonce=${share.nonce}")
            } else {
                Timber.w("Share rejected: nonce=${share.nonce}")
            }

            Result.success(accepted)
        } catch (e: Exception) {
            Timber.e(e, "Failed to submit share")
            Result.failure(e)
        }
    }

    override fun isConnected(): Boolean = isConnected.get()

    private suspend fun performSubscription(): Boolean = withContext(Dispatchers.IO) {
        try {
            val request = StratumRequest(
                id = messageId.getAndIncrement(),
                method = "mining.subscribe",
                params = listOf("ShellMiner/1.0", "EthereumStratum/1.0.0")
            )

            val response = sendRequest(request)
            response?.error == null && response?.result != null
        } catch (e: Exception) {
            Timber.e(e, "Subscription failed")
            false
        }
    }

    private suspend fun performAuthorization(): Boolean = withContext(Dispatchers.IO) {
        try {
            // Use a default wallet address for now
            // In production, this would come from user settings
            val walletAddress = "shell1mobileminer0000000000000000000000000"
            
            val request = StratumRequest(
                id = messageId.getAndIncrement(),
                method = "mining.authorize",
                params = listOf(walletAddress, "mobile_password")
            )

            val response = sendRequest(request)
            val authorized = response?.result as? Boolean ?: false
            
            if (authorized) {
                Timber.d("Successfully authorized with pool")
            } else {
                Timber.w("Pool authorization failed")
            }
            
            authorized
        } catch (e: Exception) {
            Timber.e(e, "Authorization failed")
            false
        }
    }

    private suspend fun sendRequest(request: StratumRequest): StratumResponse? = withContext(Dispatchers.IO) {
        try {
            val writer = this@PoolClientImpl.writer ?: return@withContext null
            val reader = this@PoolClientImpl.reader ?: return@withContext null

            // Send request
            val requestJson = json.encodeToString(StratumRequest.serializer(), request)
            writer.println(requestJson)
            
            // Read response
            val responseJson = reader.readLine() ?: return@withContext null
            json.decodeFromString(StratumResponse.serializer(), responseJson)
        } catch (e: Exception) {
            Timber.e(e, "Error sending request: ${request.method}")
            null
        }
    }

    private fun startConnectionMonitoring() {
        connectionJob = CoroutineScope(Dispatchers.IO).launch {
            while (isConnected.get()) {
                try {
                    // Check connection health
                    if (socket?.isConnected != true || socket?.isClosed == true) {
                        Timber.w("Pool connection lost")
                        isConnected.set(false)
                        break
                    }
                    delay(10000) // Check every 10 seconds
                } catch (e: Exception) {
                    Timber.e(e, "Connection monitoring error")
                    isConnected.set(false)
                    break
                }
            }
        }

        // Start heartbeat
        heartbeatJob = CoroutineScope(Dispatchers.IO).launch {
            while (isConnected.get()) {
                try {
                    // Send periodic ping to keep connection alive
                    val pingRequest = StratumRequest(
                        id = messageId.getAndIncrement(),
                        method = "mining.ping",
                        params = emptyList()
                    )
                    sendRequest(pingRequest)
                    delay(60000) // Ping every minute
                } catch (e: Exception) {
                    Timber.e(e, "Heartbeat error")
                    break
                }
            }
        }
    }

    @Serializable
    private data class StratumRequest(
        val id: Long,
        val method: String,
        val params: List<Any>
    )

    @Serializable
    private data class StratumResponse(
        val id: Long? = null,
        val result: Any? = null,
        val error: String? = null
    )
} 