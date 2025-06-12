package com.shell.miner.ui.theme

import android.app.Activity
import android.os.Build
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.dynamicDarkColorScheme
import androidx.compose.material3.dynamicLightColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.SideEffect
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.toArgb
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalView
import androidx.core.view.WindowCompat

// Shell Reserve brand colors
private val ShellBlue = Color(0xFF2196F3)
private val ShellDarkBlue = Color(0xFF1976D2)
private val ShellLightBlue = Color(0xFF64B5F6)
private val ShellAccent = Color(0xFF00BCD4)
private val ShellGreen = Color(0xFF4CAF50)
private val ShellOrange = Color(0xFFFF9800)
private val ShellRed = Color(0xFFF44336)

private val DarkColorScheme = darkColorScheme(
    primary = ShellBlue,
    onPrimary = Color.White,
    primaryContainer = ShellDarkBlue,
    onPrimaryContainer = Color.White,
    secondary = ShellAccent,
    onSecondary = Color.Black,
    secondaryContainer = Color(0xFF004D5C),
    onSecondaryContainer = Color.White,
    tertiary = ShellGreen,
    onTertiary = Color.Black,
    error = ShellRed,
    onError = Color.White,
    errorContainer = Color(0xFF601410),
    onErrorContainer = Color(0xFFFFDAD6),
    background = Color(0xFF121212),
    onBackground = Color(0xFFE1E2E1),
    surface = Color(0xFF1E1E1E),
    onSurface = Color(0xFFE1E2E1),
    surfaceVariant = Color(0xFF2C2C2C),
    onSurfaceVariant = Color(0xFFC4C7C5),
    outline = Color(0xFF8E918F)
)

private val LightColorScheme = lightColorScheme(
    primary = ShellBlue,
    onPrimary = Color.White,
    primaryContainer = ShellLightBlue,
    onPrimaryContainer = Color.Black,
    secondary = ShellAccent,
    onSecondary = Color.White,
    secondaryContainer = Color(0xFFB2EBF2),
    onSecondaryContainer = Color.Black,
    tertiary = ShellGreen,
    onTertiary = Color.White,
    error = ShellRed,
    onError = Color.White,
    errorContainer = Color(0xFFFFDAD6),
    onErrorContainer = Color(0xFF410002),
    background = Color(0xFFFFFBFE),
    onBackground = Color(0xFF1C1B1F),
    surface = Color(0xFFFFFBFE),
    onSurface = Color(0xFF1C1B1F),
    surfaceVariant = Color(0xFFE7E0EC),
    onSurfaceVariant = Color(0xFF49454F),
    outline = Color(0xFF79747E)
)

@Composable
fun ShellMinerTheme(
    darkTheme: Boolean = isSystemInDarkTheme(),
    // Dynamic color is available on Android 12+
    dynamicColor: Boolean = true,
    content: @Composable () -> Unit
) {
    val colorScheme = when {
        dynamicColor && Build.VERSION.SDK_INT >= Build.VERSION_CODES.S -> {
            val context = LocalContext.current
            if (darkTheme) dynamicDarkColorScheme(context) else dynamicLightColorScheme(context)
        }

        darkTheme -> DarkColorScheme
        else -> LightColorScheme
    }
    val view = LocalView.current
    if (!view.isInEditMode) {
        SideEffect {
            val window = (view.context as Activity).window
            window.statusBarColor = colorScheme.primary.toArgb()
            WindowCompat.getInsetsController(window, view).isAppearanceLightStatusBars = darkTheme
        }
    }

    MaterialTheme(
        colorScheme = colorScheme,
        typography = Typography,
        content = content
    )
} 