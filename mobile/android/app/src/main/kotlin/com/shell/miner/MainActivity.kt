package com.shell.miner

import android.Manifest
import android.content.pm.PackageManager
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.core.content.ContextCompat
import androidx.hilt.navigation.compose.hiltViewModel
import dagger.hilt.android.AndroidEntryPoint
import com.shell.miner.ui.theme.ShellMinerTheme
import com.shell.miner.ui.mining.MiningDashboard
import com.shell.miner.ui.mining.MiningViewModel
import timber.log.Timber

@AndroidEntryPoint
class MainActivity : ComponentActivity() {

    private val permissionLauncher = registerForActivityResult(
        ActivityResultContracts.RequestMultiplePermissions()
    ) { permissions ->
        when {
            permissions[Manifest.permission.WAKE_LOCK] == true -> {
                Timber.d("Wake lock permission granted")
            }
            permissions[Manifest.permission.FOREGROUND_SERVICE] == true -> {
                Timber.d("Foreground service permission granted")
            }
            else -> {
                Timber.w("Mining permissions not granted")
            }
        }
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        // Initialize native mining library
        initializeNativeMining()

        // Request necessary permissions
        requestMiningPermissions()

        setContent {
            ShellMinerTheme {
                Surface(
                    modifier = Modifier.fillMaxSize(),
                    color = MaterialTheme.colorScheme.background
                ) {
                    MiningApp()
                }
            }
        }
    }

    private fun initializeNativeMining() {
        try {
            System.loadLibrary("shellmining")
            Timber.d("Native mining library loaded successfully")
        } catch (e: UnsatisfiedLinkError) {
            Timber.e(e, "Failed to load native mining library")
        }
    }

    private fun requestMiningPermissions() {
        val permissions = arrayOf(
            Manifest.permission.WAKE_LOCK,
            Manifest.permission.FOREGROUND_SERVICE,
            Manifest.permission.INTERNET,
            Manifest.permission.ACCESS_NETWORK_STATE
        )

        val missingPermissions = permissions.filter {
            ContextCompat.checkSelfPermission(this, it) != PackageManager.PERMISSION_GRANTED
        }

        if (missingPermissions.isNotEmpty()) {
            permissionLauncher.launch(missingPermissions.toTypedArray())
        }
    }
}

@Composable
fun MiningApp() {
    val viewModel: MiningViewModel = hiltViewModel()
    val uiState by viewModel.uiState.collectAsState()

    MiningDashboard(
        uiState = uiState,
        onToggleMining = viewModel::toggleMining,
        onAdjustIntensity = viewModel::adjustIntensity,
        onUpdateSettings = viewModel::updateSettings
    )
} 