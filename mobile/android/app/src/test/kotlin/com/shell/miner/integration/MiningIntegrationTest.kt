package com.shell.miner.integration

import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import com.shell.miner.data.managers.PowerManagerImpl
import com.shell.miner.data.managers.ThermalManagerImpl
import com.shell.miner.data.repository.MiningRepositoryImpl
import com.shell.miner.data.repository.PoolClientImpl
import com.shell.miner.domain.*
import com.shell.miner.nativecode.MiningEngine
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.test.*
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import org.mockito.Mock
import org.mockito.MockitoAnnotations
import org.mockito.kotlin.*
import kotlin.test.assertEquals
import kotlin.test.assertTrue
import kotlin.test.assertFalse

@OptIn(ExperimentalCoroutinesApi::class)
@RunWith(AndroidJUnit4::class)
class MiningIntegrationTest {
    
    private lateinit var testScope: TestScope
    private lateinit var miningRepository: MiningRepositoryImpl
    
    @Mock
    private lateinit var mockMiningEngine: MiningEngine
    
    @Mock
    private lateinit var mockPoolClient: PoolClientImpl
    
    @Mock
    private lateinit var mockPowerManager: PowerManagerImpl
    
    @Mock
    private lateinit var mockThermalManager: ThermalManagerImpl
    
    @Before
    fun setup() {
        MockitoAnnotations.openMocks(this)
        testScope = TestScope()
        
        miningRepository = MiningRepositoryImpl(
            miningEngine = mockMiningEngine,
            poolClient = mockPoolClient,
            powerManager = mockPowerManager,
            thermalManager = mockThermalManager,
            scope = testScope
        )
    }
    
    @Test
    fun `test complete mining workflow - success scenario`() = testScope.runTest {
        // Given: Optimal conditions for mining
        whenever(mockPowerManager.shouldStartMining(any())).thenReturn(true)
        whenever(mockPowerManager.determineOptimalIntensity(any())).thenReturn(MiningIntensity.MEDIUM)
        whenever(mockThermalManager.getCurrentTemperature()).thenReturn(35.0f)
        whenever(mockThermalManager.canMineAtIntensity(any())).thenReturn(true)
        whenever(mockMiningEngine.initializeEngine()).thenReturn(true)
        whenever(mockMiningEngine.startMining(any())).thenReturn(true)
        whenever(mockPoolClient.connect(any())).thenReturn(Result.success(Unit))
        
        // When: Starting mining
        val config = MiningConfig(
            poolUrl = "stratum+tcp://pool.shellreserve.com:4444",
            walletAddress = "xsl1qtest123...",
            intensity = MiningIntensity.AUTO,
            enableNPU = true,
            thermalLimit = 45.0f
        )
        
        val result = miningRepository.startMining(config)
        
        // Then: Mining should start successfully
        assertTrue(result.isSuccess)
        
        // Verify: All components were initialized in correct order
        inOrder(mockPowerManager, mockThermalManager, mockMiningEngine, mockPoolClient) {
            verify(mockPowerManager).shouldStartMining(config)
            verify(mockThermalManager).canMineAtIntensity(MiningIntensity.MEDIUM)
            verify(mockMiningEngine).initializeEngine()
            verify(mockPoolClient).connect("stratum+tcp://pool.shellreserve.com:4444")
            verify(mockMiningEngine).startMining(MiningIntensity.MEDIUM)
        }
        
        // Verify: Mining state is properly tracked
        val miningState = miningRepository.getMiningState().first()
        assertTrue(miningState.isMining)
        assertEquals(MiningIntensity.MEDIUM, miningState.intensity)
    }
    
    @Test
    fun `test mining workflow with power constraints`() = testScope.runTest {
        // Given: Low battery conditions
        whenever(mockPowerManager.shouldStartMining(any())).thenReturn(false)
        whenever(mockPowerManager.getPowerConstraints()).thenReturn(
            PowerConstraints(
                batteryLevel = 75,
                isCharging = false,
                reason = PowerConstraintReason.LOW_BATTERY
            )
        )
        
        val config = MiningConfig(
            poolUrl = "stratum+tcp://pool.shellreserve.com:4444",
            walletAddress = "xsl1qtest123...",
            intensity = MiningIntensity.FULL
        )
        
        // When: Attempting to start mining
        val result = miningRepository.startMining(config)
        
        // Then: Mining should fail due to power constraints
        assertTrue(result.isFailure)
        assertEquals("Power conditions not met: LOW_BATTERY", result.exceptionOrNull()?.message)
        
        // Verify: Mining engine was never initialized
        verify(mockMiningEngine, never()).initializeEngine()
        verify(mockMiningEngine, never()).startMining(any())
    }
    
    @Test
    fun `test thermal throttling during mining`() = testScope.runTest {
        // Given: Mining is active but temperature rises
        whenever(mockThermalManager.getCurrentTemperature()).thenReturn(48.0f) // High temp
        whenever(mockThermalManager.canMineAtIntensity(any())).thenReturn(false)
        whenever(mockThermalManager.getSuggestedIntensity()).thenReturn(MiningIntensity.LIGHT)
        whenever(mockMiningEngine.adjustIntensity(any())).thenReturn(true)
        
        // Simulate thermal event
        miningRepository.handleThermalEvent(
            ThermalEvent(
                temperature = 48.0f,
                thermalState = ThermalState.SEVERE,
                suggestedAction = ThermalAction.THROTTLE_SEVERELY
            )
        )
        
        // Then: Mining should be throttled
        verify(mockMiningEngine).adjustIntensity(MiningIntensity.LIGHT)
        
        // Verify: State reflects thermal throttling
        val miningState = miningRepository.getMiningState().first()
        assertTrue(miningState.isThermalThrottled)
        assertEquals(48.0f, miningState.temperature)
    }
    
    @Test
    fun `test NPU utilization and fallback`() = testScope.runTest {
        // Given: NPU is available but fails during initialization
        whenever(mockMiningEngine.isNPUAvailable()).thenReturn(true)
        whenever(mockMiningEngine.initializeNPU()).thenReturn(false) // NPU init fails
        whenever(mockMiningEngine.fallbackToCPU()).thenReturn(true)
        
        val config = MiningConfig(
            enableNPU = true,
            intensity = MiningIntensity.FULL
        )
        
        // When: Starting mining with NPU enabled
        miningRepository.configureNPU(config)
        
        // Then: Should fallback to CPU mining
        verify(mockMiningEngine).initializeNPU()
        verify(mockMiningEngine).fallbackToCPU()
        
        // Verify: State reflects CPU fallback
        val miningState = miningRepository.getMiningState().first()
        assertFalse(miningState.isNPUActive)
        assertTrue(miningState.isCPUFallback)
    }
    
    @Test
    fun `test pool connection and share submission`() = testScope.runTest {
        // Given: Pool client is connected and receives work
        val mockWorkTemplate = WorkTemplate(
            blockHeight = 123456,
            previousBlockHash = "000000abc123...",
            merkleRoot = "def456...",
            target = "00000000ffff...",
            nonce = 0,
            thermalTarget = "0000ffff..."
        )
        
        whenever(mockPoolClient.getWork()).thenReturn(Result.success(mockWorkTemplate))
        whenever(mockMiningEngine.mineWork(any())).thenReturn(
            MiningResult(
                nonce = 42,
                hash = "000000123abc...",
                thermalProof = "thermal_proof_data",
                npuMetrics = NPUMetrics(utilization = 85.5f, operations = 1234)
            )
        )
        whenever(mockPoolClient.submitShare(any())).thenReturn(Result.success(true))
        
        // When: Mining finds a valid share
        val shareResult = miningRepository.processWorkTemplate(mockWorkTemplate)
        
        // Then: Share should be submitted successfully
        assertTrue(shareResult.isSuccess)
        
        // Verify: Share includes mobile-specific data
        argumentCaptor<MiningShare>().apply {
            verify(mockPoolClient).submitShare(capture())
            val submittedShare = firstValue
            
            assertEquals(42, submittedShare.nonce)
            assertEquals("000000123abc...", submittedShare.hash)
            assertEquals("thermal_proof_data", submittedShare.thermalProof)
            assertTrue(submittedShare.npuMetrics != null)
            assertEquals(85.5f, submittedShare.npuMetrics!!.utilization)
        }
    }
    
    @Test
    fun `test device capability detection`() = testScope.runTest {
        // Given: Device capabilities are detected
        val deviceCapabilities = DeviceCapabilities(
            deviceClass = DeviceClass.FLAGSHIP,
            socModel = "Snapdragon 8 Gen 3",
            coreCount = 8,
            hasNPU = true,
            npuModel = "Hexagon DSP",
            maxThermalBudget = 8.0f,
            recommendedIntensity = MiningIntensity.FULL
        )
        
        whenever(mockMiningEngine.detectDeviceCapabilities()).thenReturn(deviceCapabilities)
        
        // When: Initializing the mining repository
        val capabilities = miningRepository.getDeviceCapabilities()
        
        // Then: Device should be properly classified
        assertEquals(DeviceClass.FLAGSHIP, capabilities.deviceClass)
        assertTrue(capabilities.hasNPU)
        assertEquals(MiningIntensity.FULL, capabilities.recommendedIntensity)
        
        // Verify: Mining configuration is optimized for device
        val optimizedConfig = miningRepository.getOptimizedConfig(capabilities)
        assertEquals(MiningIntensity.FULL, optimizedConfig.intensity)
        assertTrue(optimizedConfig.enableNPU)
        assertEquals(45.0f, optimizedConfig.thermalLimit) // Flagship thermal limit
    }
    
    @Test
    fun `test error recovery and resilience`() = testScope.runTest {
        // Given: Various error conditions occur
        whenever(mockPoolClient.connect(any())).thenReturn(Result.failure(Exception("Network error")))
        whenever(mockMiningEngine.startMining(any())).thenReturn(false)
        
        val config = MiningConfig(
            poolUrl = "stratum+tcp://pool.shellreserve.com:4444",
            retryAttempts = 3,
            retryDelay = 100L
        )
        
        // When: Starting mining with network errors
        val result = miningRepository.startMiningWithRetry(config)
        
        // Then: Should retry connection attempts
        verify(mockPoolClient, times(3)).connect(any())
        assertTrue(result.isFailure)
        
        // Verify: Error state is properly tracked
        val miningState = miningRepository.getMiningState().first()
        assertFalse(miningState.isMining)
        assertTrue(miningState.hasError)
        assertEquals("Network error", miningState.errorMessage)
    }
    
    @Test
    fun `test performance metrics collection`() = testScope.runTest {
        // Given: Mining is active and collecting metrics
        val performanceMetrics = PerformanceMetrics(
            hashRate = 125.5,
            powerConsumption = 6.2f,
            temperature = 42.1f,
            npuUtilization = 78.3f,
            coreUtilization = mapOf(
                0 to 95.0f, // Big core 1
                1 to 90.0f, // Big core 2
                2 to 60.0f, // Little core 1
                3 to 55.0f  // Little core 2
            ),
            thermalThrottleEvents = 2,
            uptime = 3600L // 1 hour
        )
        
        whenever(mockMiningEngine.getPerformanceMetrics()).thenReturn(performanceMetrics)
        
        // When: Collecting performance data
        val metrics = miningRepository.getPerformanceMetrics()
        
        // Then: All metrics should be properly collected
        assertEquals(125.5, metrics.hashRate)
        assertEquals(6.2f, metrics.powerConsumption)
        assertEquals(42.1f, metrics.temperature)
        assertEquals(78.3f, metrics.npuUtilization)
        assertEquals(2, metrics.thermalThrottleEvents)
        
        // Verify: Core utilization is tracked per core type
        assertTrue(metrics.coreUtilization[0]!! > 90.0f) // Big cores are highly utilized
        assertTrue(metrics.coreUtilization[2]!! < 70.0f) // Little cores are moderately utilized
    }
}

// Supporting data classes for integration tests
data class PowerConstraints(
    val batteryLevel: Int,
    val isCharging: Boolean,
    val reason: PowerConstraintReason
)

enum class PowerConstraintReason {
    LOW_BATTERY,
    NOT_CHARGING,
    THERMAL_LIMIT,
    USER_DISABLED
}

data class ThermalEvent(
    val temperature: Float,
    val thermalState: ThermalState,
    val suggestedAction: ThermalAction
)

enum class ThermalState {
    NORMAL,
    LIGHT,
    MODERATE,
    SEVERE,
    CRITICAL
}

enum class ThermalAction {
    CONTINUE,
    THROTTLE_LIGHTLY,
    THROTTLE_MODERATELY,
    THROTTLE_SEVERELY,
    STOP_MINING
}

data class WorkTemplate(
    val blockHeight: Long,
    val previousBlockHash: String,
    val merkleRoot: String,
    val target: String,
    val nonce: Long,
    val thermalTarget: String
)

data class MiningResult(
    val nonce: Long,
    val hash: String,
    val thermalProof: String,
    val npuMetrics: NPUMetrics?
)

data class NPUMetrics(
    val utilization: Float,
    val operations: Long
)

data class MiningShare(
    val nonce: Long,
    val hash: String,
    val thermalProof: String,
    val npuMetrics: NPUMetrics?
)

data class DeviceCapabilities(
    val deviceClass: DeviceClass,
    val socModel: String,
    val coreCount: Int,
    val hasNPU: Boolean,
    val npuModel: String,
    val maxThermalBudget: Float,
    val recommendedIntensity: MiningIntensity
)

enum class DeviceClass {
    BUDGET,
    MIDRANGE,
    FLAGSHIP
}

data class PerformanceMetrics(
    val hashRate: Double,
    val powerConsumption: Float,
    val temperature: Float,
    val npuUtilization: Float,
    val coreUtilization: Map<Int, Float>,
    val thermalThrottleEvents: Int,
    val uptime: Long
) 