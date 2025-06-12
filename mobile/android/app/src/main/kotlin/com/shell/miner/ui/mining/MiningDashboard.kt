package com.shell.miner.ui.mining

import androidx.compose.animation.*
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.shell.miner.domain.MiningConfig
import com.shell.miner.domain.MiningIntensity
import com.shell.miner.domain.ThermalState

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun MiningDashboard(
    uiState: MiningUiState,
    onToggleMining: () -> Unit,
    onAdjustIntensity: (MiningIntensity) -> Unit,
    onUpdateSettings: (MiningConfig) -> Unit,
    modifier: Modifier = Modifier
) {
    Column(
        modifier = modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp)
    ) {
        // Header with status
        MiningHeader(uiState = uiState)
        
        // Warning card if needed
        AnimatedVisibility(
            visible = uiState.error != null,
            enter = slideInVertically() + fadeIn(),
            exit = slideOutVertically() + fadeOut()
        ) {
            uiState.error?.let { ErrorCard(message = it) }
        }
        
        // Main mining stats card
        MiningStatsCard(uiState = uiState)
        
        // Power and thermal status
        PowerThermalCard(uiState = uiState)
        
        // Mining controls
        MiningControlsCard(
            uiState = uiState,
            onToggleMining = onToggleMining,
            onAdjustIntensity = onAdjustIntensity
        )
        
        // Performance details
        PerformanceDetailsCard(uiState = uiState)
        
        // Earnings card
        EarningsCard(uiState = uiState)
    }
}

@Composable
private fun MiningHeader(uiState: MiningUiState) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(
            containerColor = if (uiState.isMining) 
                MaterialTheme.colorScheme.primaryContainer 
            else 
                MaterialTheme.colorScheme.surfaceVariant
        )
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(20.dp),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically
        ) {
            Column {
                Text(
                    text = "Shell Reserve Mining",
                    style = MaterialTheme.typography.headlineSmall,
                    fontWeight = FontWeight.Bold
                )
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    StatusIndicator(isActive = uiState.isMining)
                    Text(
                        text = if (uiState.isMining) "Mining Active" else "Mining Stopped",
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
            }
            
            Icon(
                imageVector = if (uiState.isMining) Icons.Default.PlayArrow else Icons.Default.Stop,
                contentDescription = null,
                modifier = Modifier.size(32.dp),
                tint = MaterialTheme.colorScheme.primary
            )
        }
    }
}

@Composable
private fun StatusIndicator(isActive: Boolean) {
    Box(
        modifier = Modifier
            .size(12.dp)
            .clip(RoundedCornerShape(6.dp))
            .background(
                if (isActive) Color.Green else Color.Gray
            )
    )
}

@Composable
private fun ErrorCard(message: String) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.errorContainer
        )
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp),
            horizontalArrangement = Arrangement.spacedBy(12.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Icon(
                imageVector = Icons.Default.Warning,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.error
            )
            Text(
                text = message,
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onErrorContainer
            )
        }
    }
}

@Composable
private fun MiningStatsCard(uiState: MiningUiState) {
    Card(
        modifier = Modifier.fillMaxWidth()
    ) {
        Column(
            modifier = Modifier.padding(20.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            Text(
                text = "Mining Statistics",
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.SemiBold
            )
            
            // Hash rate display
            HashRateDisplay(
                totalHashRate = uiState.hashRate,
                randomXHashRate = uiState.randomXHashRate,
                mobileXHashRate = uiState.mobileXHashRate
            )
            
            // Share and block stats
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceEvenly
            ) {
                StatItem(
                    label = "Shares",
                    value = uiState.sharesSubmitted.toString(),
                    icon = Icons.Default.Share
                )
                StatItem(
                    label = "Blocks",
                    value = uiState.blocksFound.toString(),
                    icon = Icons.Default.Inventory
                )
                StatItem(
                    label = "NPU",
                    value = "${(uiState.npuUtilization * 100).toInt()}%",
                    icon = Icons.Default.Memory
                )
            }
        }
    }
}

@Composable
private fun HashRateDisplay(
    totalHashRate: Double,
    randomXHashRate: Double,
    mobileXHashRate: Double
) {
    Column(
        verticalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        // Total hash rate
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.Bottom
        ) {
            Text(
                text = "Total Hash Rate",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
            Text(
                text = formatHashRate(totalHashRate),
                style = MaterialTheme.typography.headlineSmall,
                fontWeight = FontWeight.Bold,
                color = MaterialTheme.colorScheme.primary
            )
        }
        
        // Algorithm breakdown
        if (randomXHashRate > 0 || mobileXHashRate > 0) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween
            ) {
                Text(
                    text = "RandomX: ${formatHashRate(randomXHashRate)}",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
                Text(
                    text = "MobileX: ${formatHashRate(mobileXHashRate)}",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
        }
    }
}

@Composable
private fun StatItem(
    label: String,
    value: String,
    icon: ImageVector
) {
    Column(
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(4.dp)
    ) {
        Icon(
            imageVector = icon,
            contentDescription = null,
            tint = MaterialTheme.colorScheme.primary,
            modifier = Modifier.size(20.dp)
        )
        Text(
            text = value,
            style = MaterialTheme.typography.titleMedium,
            fontWeight = FontWeight.Bold
        )
        Text(
            text = label,
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )
    }
}

@Composable
private fun PowerThermalCard(uiState: MiningUiState) {
    Card(
        modifier = Modifier.fillMaxWidth()
    ) {
        Column(
            modifier = Modifier.padding(20.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            Text(
                text = "Device Status",
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.SemiBold
            )
            
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceEvenly
            ) {
                // Battery status
                DeviceStatusItem(
                    icon = if (uiState.isCharging) Icons.Default.BatteryChargingFull else Icons.Default.Battery6Bar,
                    label = "Battery",
                    value = "${uiState.batteryLevel}%",
                    color = getBatteryColor(uiState.batteryLevel, uiState.isCharging)
                )
                
                // Temperature status
                DeviceStatusItem(
                    icon = Icons.Default.DeviceThermostat,
                    label = "Temperature",
                    value = "${uiState.temperature.toInt()}°C",
                    color = getThermalColor(uiState.thermalState)
                )
                
                // Mining intensity
                DeviceStatusItem(
                    icon = Icons.Default.Speed,
                    label = "Intensity",
                    value = uiState.intensity.displayName,
                    color = MaterialTheme.colorScheme.primary
                )
            }
        }
    }
}

@Composable
private fun DeviceStatusItem(
    icon: ImageVector,
    label: String,
    value: String,
    color: Color
) {
    Column(
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(4.dp)
    ) {
        Icon(
            imageVector = icon,
            contentDescription = null,
            tint = color,
            modifier = Modifier.size(24.dp)
        )
        Text(
            text = value,
            style = MaterialTheme.typography.titleMedium,
            fontWeight = FontWeight.Bold,
            color = color
        )
        Text(
            text = label,
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )
    }
}

@Composable
private fun MiningControlsCard(
    uiState: MiningUiState,
    onToggleMining: () -> Unit,
    onAdjustIntensity: (MiningIntensity) -> Unit
) {
    Card(
        modifier = Modifier.fillMaxWidth()
    ) {
        Column(
            modifier = Modifier.padding(20.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            Text(
                text = "Mining Controls",
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.SemiBold
            )
            
            // Main toggle button
            Button(
                onClick = onToggleMining,
                modifier = Modifier
                    .fillMaxWidth()
                    .height(56.dp),
                colors = ButtonDefaults.buttonColors(
                    containerColor = if (uiState.isMining) 
                        MaterialTheme.colorScheme.error 
                    else 
                        MaterialTheme.colorScheme.primary
                )
            ) {
                Icon(
                    imageVector = if (uiState.isMining) Icons.Default.Stop else Icons.Default.PlayArrow,
                    contentDescription = null,
                    modifier = Modifier.size(20.dp)
                )
                Spacer(modifier = Modifier.width(8.dp))
                Text(
                    text = if (uiState.isMining) "Stop Mining" else "Start Mining",
                    style = MaterialTheme.typography.titleMedium
                )
            }
            
            // Intensity selection
            Text(
                text = "Mining Intensity",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
            
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp)
            ) {
                MiningIntensity.values().filter { it != MiningIntensity.DISABLED }.forEach { intensity ->
                    IntensityChip(
                        intensity = intensity,
                        isSelected = uiState.intensity == intensity,
                        onClick = { onAdjustIntensity(intensity) },
                        modifier = Modifier.weight(1f)
                    )
                }
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun IntensityChip(
    intensity: MiningIntensity,
    isSelected: Boolean,
    onClick: () -> Unit,
    modifier: Modifier = Modifier
) {
    FilterChip(
        onClick = onClick,
        label = {
            Text(
                text = intensity.displayName,
                textAlign = TextAlign.Center,
                modifier = Modifier.fillMaxWidth()
            )
        },
        selected = isSelected,
        modifier = modifier
    )
}

@Composable
private fun PerformanceDetailsCard(uiState: MiningUiState) {
    Card(
        modifier = Modifier.fillMaxWidth()
    ) {
        Column(
            modifier = Modifier.padding(20.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            Text(
                text = "Performance Details",
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.SemiBold
            )
            
            PerformanceRow(
                label = "Algorithm",
                value = uiState.algorithm.displayName
            )
            
            PerformanceRow(
                label = "NPU Utilization",
                value = "${(uiState.npuUtilization * 100).toInt()}%"
            )
            
            PerformanceRow(
                label = "Thermal Throttling",
                value = if (uiState.thermalThrottling) "Active" else "None"
            )
            
            PerformanceRow(
                label = "Power State",
                value = if (uiState.isCharging) "Charging" else "Battery"
            )
        }
    }
}

@Composable
private fun PerformanceRow(
    label: String,
    value: String
) {
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.SpaceBetween
    ) {
        Text(
            text = label,
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )
        Text(
            text = value,
            style = MaterialTheme.typography.bodyMedium,
            fontWeight = FontWeight.Medium
        )
    }
}

@Composable
private fun EarningsCard(uiState: MiningUiState) {
    Card(
        modifier = Modifier.fillMaxWidth()
    ) {
        Column(
            modifier = Modifier.padding(20.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            Text(
                text = "Estimated Earnings",
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.SemiBold
            )
            
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween
            ) {
                Column {
                    Text(
                        text = "Current Session",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                    Text(
                        text = String.format("%.6f SHELL", uiState.estimatedEarnings),
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold,
                        color = MaterialTheme.colorScheme.primary
                    )
                }
                
                Column(
                    horizontalAlignment = Alignment.End
                ) {
                    Text(
                        text = "Projected Daily",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                    Text(
                        text = String.format("%.6f SHELL", uiState.projectedDailyEarnings),
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold,
                        color = MaterialTheme.colorScheme.secondary
                    )
                }
            }
        }
    }
}

// Helper functions
private fun formatHashRate(hashRate: Double): String {
    return when {
        hashRate >= 1_000_000 -> String.format("%.2f MH/s", hashRate / 1_000_000)
        hashRate >= 1_000 -> String.format("%.2f KH/s", hashRate / 1_000)
        else -> String.format("%.2f H/s", hashRate)
    }
}

@Composable
private fun getBatteryColor(level: Int, isCharging: Boolean): Color {
    return when {
        isCharging -> Color(0xFF4CAF50) // Green
        level > 50 -> Color(0xFF4CAF50) // Green
        level > 20 -> Color(0xFFFF9800) // Orange
        else -> Color(0xFFF44336) // Red
    }
}

@Composable
private fun getThermalColor(thermalState: ThermalState): Color {
    return when (thermalState) {
        ThermalState.NOMINAL -> Color(0xFF4CAF50) // Green
        ThermalState.FAIR -> Color(0xFFFF9800) // Orange
        ThermalState.SERIOUS -> Color(0xFFFF5722) // Deep Orange
        ThermalState.CRITICAL -> Color(0xFFF44336) // Red
    }
} 