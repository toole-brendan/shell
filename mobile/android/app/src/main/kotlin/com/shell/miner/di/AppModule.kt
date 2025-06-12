package com.shell.miner.di

import android.content.Context
import com.shell.miner.data.managers.PowerManagerImpl
import com.shell.miner.data.managers.ThermalManagerImpl
import com.shell.miner.data.repository.MiningRepositoryImpl
import com.shell.miner.data.repository.PoolClientImpl
import com.shell.miner.domain.*
import com.shell.miner.nativecode.MiningEngine
import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.android.qualifiers.ApplicationContext
import dagger.hilt.components.SingletonComponent
import javax.inject.Singleton

@Module
@InstallIn(SingletonComponent::class)
object AppModule {

    @Provides
    @Singleton
    fun provideMiningEngine(): MiningEngine {
        return MiningEngine()
    }

    @Provides
    @Singleton
    fun provideThermalManager(): ThermalManager {
        return ThermalManagerImpl()
    }

    @Provides
    @Singleton
    fun providePowerManager(
        @ApplicationContext context: Context,
        thermalManager: ThermalManager
    ): PowerManager {
        return PowerManagerImpl(context, thermalManager)
    }

    @Provides
    @Singleton
    fun providePoolClient(): PoolClient {
        return PoolClientImpl()
    }

    @Provides
    @Singleton
    fun provideMiningRepository(
        miningEngine: MiningEngine,
        powerManager: PowerManager,
        thermalManager: ThermalManager,
        poolClient: PoolClient
    ): MiningRepository {
        return MiningRepositoryImpl(
            miningEngine = miningEngine,
            powerManager = powerManager,
            thermalManager = thermalManager,
            poolClient = poolClient
        )
    }
} 