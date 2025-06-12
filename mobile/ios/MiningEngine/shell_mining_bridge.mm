#import "shell_mining_bridge.h"
#import "ios_mobile_randomx.h"
#import "ios_thermal_manager.h"
#import "core_ml_npu_provider.h"
#import <UIKit/UIKit.h>
#import <IOKit/IOKitLib.h>
#import <CoreML/CoreML.h>

// Native implementations
@implementation NativeMiningStats
@end

@implementation NativeMiningConfig
@end

@implementation NativePowerState
@end

@implementation NativeThermalState
@end

// Main bridge implementation
@implementation ShellMiningBridge {
    // C++ mining engine
    std::unique_ptr<shell::mobile::ios::IOSMobileXMiner> _miner;
    std::unique_ptr<shell::mobile::ios::IOSThermalManager> _thermalManager;
    std::unique_ptr<shell::mobile::ios::CoreMLNPUProvider> _npuProvider;
    
    // State tracking
    BOOL _isInitialized;
    BOOL _isMining;
    NativeMiningConfig *_currentConfig;
    
    // Timer for stats updates
    NSTimer *_statsTimer;
    dispatch_queue_t _miningQueue;
}

#pragma mark - Initialization

- (instancetype)init {
    self = [super init];
    if (self) {
        _isInitialized = NO;
        _isMining = NO;
        _currentConfig = nil;
        
        // Create dedicated queue for mining operations
        _miningQueue = dispatch_queue_create("com.shell.mining", DISPATCH_QUEUE_CONCURRENT);
        
        // Initialize C++ components
        try {
            _thermalManager = std::make_unique<shell::mobile::ios::IOSThermalManager>();
            _npuProvider = std::make_unique<shell::mobile::ios::CoreMLNPUProvider>();
            _miner = std::make_unique<shell::mobile::ios::IOSMobileXMiner>();
        } catch (const std::exception& e) {
            NSLog(@"Failed to initialize C++ components: %s", e.what());
            return nil;
        }
    }
    return self;
}

- (void)dealloc {
    [self shutdown];
}

#pragma mark - Mining Lifecycle

- (BOOL)initializeWithConfig:(NativeMiningConfig *)config error:(NSError **)error {
    if (_isInitialized) {
        if (error) {
            *error = [NSError errorWithDomain:@"ShellMining" 
                                        code:1001 
                                    userInfo:@{NSLocalizedDescriptionKey: @"Already initialized"}];
        }
        return NO;
    }
    
    @try {
        // Convert Swift config to C++ config
        shell::mobile::ios::IOSMiningConfig cppConfig;
        cppConfig.intensity = static_cast<int>(config.intensity);
        cppConfig.algorithm = static_cast<int>(config.algorithm);
        cppConfig.npuEnabled = config.npuEnabled;
        cppConfig.maxTemperature = config.maxTemperature;
        cppConfig.throttleTemperature = config.throttleTemperature;
        cppConfig.coreCount = static_cast<uint32_t>(config.coreCount);
        
        // Initialize thermal manager
        if (!_thermalManager->initialize()) {
            if (error) {
                *error = [NSError errorWithDomain:@"ShellMining" 
                                            code:1002 
                                        userInfo:@{NSLocalizedDescriptionKey: @"Failed to initialize thermal manager"}];
            }
            return NO;
        }
        
        // Initialize NPU provider if enabled
        if (config.npuEnabled) {
            if (!_npuProvider->initialize()) {
                NSLog(@"Warning: NPU initialization failed, continuing with CPU fallback");
            }
        }
        
        // Initialize mining engine
        if (!_miner->initialize(cppConfig)) {
            if (error) {
                *error = [NSError errorWithDomain:@"ShellMining" 
                                            code:1003 
                                        userInfo:@{NSLocalizedDescriptionKey: @"Failed to initialize mining engine"}];
            }
            return NO;
        }
        
        _currentConfig = config;
        _isInitialized = YES;
        
        return YES;
        
    } @catch (NSException *exception) {
        if (error) {
            *error = [NSError errorWithDomain:@"ShellMining" 
                                        code:1004 
                                    userInfo:@{NSLocalizedDescriptionKey: exception.reason}];
        }
        return NO;
    }
}

- (BOOL)startMining:(NSError **)error {
    if (!_isInitialized) {
        if (error) {
            *error = [NSError errorWithDomain:@"ShellMining" 
                                        code:1005 
                                    userInfo:@{NSLocalizedDescriptionKey: @"Not initialized"}];
        }
        return NO;
    }
    
    if (_isMining) {
        return YES; // Already mining
    }
    
    // Check thermal state before starting
    NativeThermalState *thermalState = [self getCurrentThermalState];
    if (thermalState.temperature > _currentConfig.maxTemperature) {
        if (error) {
            *error = [NSError errorWithDomain:@"ShellMining" 
                                        code:1006 
                                    userInfo:@{NSLocalizedDescriptionKey: @"Device too hot to start mining"}];
        }
        return NO;
    }
    
    // Check battery state
    NativePowerState *powerState = [self getCurrentPowerState];
    if (powerState.batteryLevel < 20 || (!powerState.isCharging && powerState.batteryLevel < 80)) {
        if (error) {
            *error = [NSError errorWithDomain:@"ShellMining" 
                                        code:1007 
                                    userInfo:@{NSLocalizedDescriptionKey: @"Insufficient battery to start mining"}];
        }
        return NO;
    }
    
    @try {
        // Start mining in background queue
        dispatch_async(_miningQueue, ^{
            if (self->_miner->startMining()) {
                dispatch_async(dispatch_get_main_queue(), ^{
                    self->_isMining = YES;
                    [self startStatsTimer];
                });
            } else {
                dispatch_async(dispatch_get_main_queue(), ^{
                    if (self.errorCallback) {
                        NSError *miningError = [NSError errorWithDomain:@"ShellMining" 
                                                                  code:1008 
                                                              userInfo:@{NSLocalizedDescriptionKey: @"Failed to start mining"}];
                        self.errorCallback(miningError);
                    }
                });
            }
        });
        
        return YES;
        
    } @catch (NSException *exception) {
        if (error) {
            *error = [NSError errorWithDomain:@"ShellMining" 
                                        code:1009 
                                    userInfo:@{NSLocalizedDescriptionKey: exception.reason}];
        }
        return NO;
    }
}

- (BOOL)stopMining:(NSError **)error {
    if (!_isMining) {
        return YES; // Already stopped
    }
    
    @try {
        [self stopStatsTimer];
        
        dispatch_async(_miningQueue, ^{
            self->_miner->stopMining();
            dispatch_async(dispatch_get_main_queue(), ^{
                self->_isMining = NO;
            });
        });
        
        return YES;
        
    } @catch (NSException *exception) {
        if (error) {
            *error = [NSError errorWithDomain:@"ShellMining" 
                                        code:1010 
                                    userInfo:@{NSLocalizedDescriptionKey: exception.reason}];
        }
        return NO;
    }
}

- (void)shutdown {
    if (_isMining) {
        [self stopMining:nil];
    }
    
    [self stopStatsTimer];
    
    if (_miner) {
        _miner->shutdown();
        _miner.reset();
    }
    
    if (_thermalManager) {
        _thermalManager.reset();
    }
    
    if (_npuProvider) {
        _npuProvider.reset();
    }
    
    _isInitialized = NO;
}

#pragma mark - Configuration

- (BOOL)updateConfig:(NativeMiningConfig *)config error:(NSError **)error {
    if (!_isInitialized) {
        if (error) {
            *error = [NSError errorWithDomain:@"ShellMining" 
                                        code:1011 
                                    userInfo:@{NSLocalizedDescriptionKey: @"Not initialized"}];
        }
        return NO;
    }
    
    @try {
        // Convert to C++ config
        shell::mobile::ios::IOSMiningConfig cppConfig;
        cppConfig.intensity = static_cast<int>(config.intensity);
        cppConfig.algorithm = static_cast<int>(config.algorithm);
        cppConfig.npuEnabled = config.npuEnabled;
        cppConfig.maxTemperature = config.maxTemperature;
        cppConfig.throttleTemperature = config.throttleTemperature;
        cppConfig.coreCount = static_cast<uint32_t>(config.coreCount);
        
        if (_miner->updateConfig(cppConfig)) {
            _currentConfig = config;
            return YES;
        } else {
            if (error) {
                *error = [NSError errorWithDomain:@"ShellMining" 
                                            code:1012 
                                        userInfo:@{NSLocalizedDescriptionKey: @"Failed to update configuration"}];
            }
            return NO;
        }
        
    } @catch (NSException *exception) {
        if (error) {
            *error = [NSError errorWithDomain:@"ShellMining" 
                                        code:1013 
                                    userInfo:@{NSLocalizedDescriptionKey: exception.reason}];
        }
        return NO;
    }
}

- (void)configureNPUWithModel:(MLModel *)model {
    if (_npuProvider && model) {
        _npuProvider->setMLModel((__bridge void*)model);
    }
}

#pragma mark - Status and Metrics

- (NativeMiningStats *)getCurrentStats {
    if (!_isInitialized) {
        return [self emptyStats];
    }
    
    @try {
        auto cppStats = _miner->getCurrentStats();
        
        NativeMiningStats *stats = [[NativeMiningStats alloc] init];
        stats.totalHashRate = cppStats.totalHashRate;
        stats.randomXHashRate = cppStats.randomXHashRate;
        stats.mobileXHashRate = cppStats.mobileXHashRate;
        stats.sharesSubmitted = cppStats.sharesSubmitted;
        stats.blocksFound = cppStats.blocksFound;
        stats.npuUtilization = cppStats.npuUtilization;
        stats.currentIntensity = static_cast<NativeMiningIntensity>(cppStats.currentIntensity);
        stats.currentAlgorithm = static_cast<NativeMiningAlgorithm>(cppStats.currentAlgorithm);
        stats.timestamp = [NSDate date];
        
        return stats;
        
    } @catch (NSException *exception) {
        NSLog(@"Error getting stats: %@", exception.reason);
        return [self emptyStats];
    }
}

- (BOOL)isMining {
    return _isMining;
}

- (uint64_t)getTotalHashes {
    if (!_isInitialized) {
        return 0;
    }
    
    @try {
        return _miner->getTotalHashes();
    } @catch (NSException *exception) {
        NSLog(@"Error getting total hashes: %@", exception.reason);
        return 0;
    }
}

#pragma mark - Power and Thermal Management

- (NativePowerState *)getCurrentPowerState {
    NativePowerState *state = [[NativePowerState alloc] init];
    
    // Get battery level and charging state
    UIDevice *device = [UIDevice currentDevice];
    device.batteryMonitoringEnabled = YES;
    
    state.batteryLevel = (int32_t)(device.batteryLevel * 100);
    state.isCharging = (device.batteryState == UIDeviceBatteryStateCharging || 
                       device.batteryState == UIDeviceBatteryStateFull);
    
    // Get power save mode
    state.isPowerSaveMode = [[NSProcessInfo processInfo] isLowPowerModeEnabled];
    
    // Get thermal state
    state.thermalState = (NSInteger)[[NSProcessInfo processInfo] thermalState];
    
    state.timestamp = [NSDate date];
    
    return state;
}

- (NativeThermalState *)getCurrentThermalState {
    NativeThermalState *state = [[NativeThermalState alloc] init];
    
    if (_thermalManager) {
        @try {
            auto cppState = _thermalManager->getCurrentState();
            state.temperature = cppState.temperature;
            state.state = cppState.state;
            state.isThrottling = cppState.isThrottling;
        } @catch (NSException *exception) {
            NSLog(@"Error getting thermal state: %@", exception.reason);
            state.temperature = 35.0f; // Default safe value
            state.state = 0; // Normal
            state.isThrottling = NO;
        }
    }
    
    state.timestamp = [NSDate date];
    
    return state;
}

- (BOOL)canMineAtIntensity:(NativeMiningIntensity)intensity {
    NativePowerState *powerState = [self getCurrentPowerState];
    NativeThermalState *thermalState = [self getCurrentThermalState];
    
    // Check battery requirements
    if (powerState.batteryLevel < 20) {
        return NO;
    }
    
    if (!powerState.isCharging && powerState.batteryLevel < 80) {
        return NO;
    }
    
    // Check thermal requirements
    if (thermalState.temperature > _currentConfig.maxTemperature) {
        return NO;
    }
    
    if (powerState.thermalState == (NSInteger)NSProcessInfoThermalStateCritical) {
        return NO;
    }
    
    return YES;
}

#pragma mark - Mining Operations

- (NSData *)computeHashForHeader:(NSData *)headerBytes algorithm:(NativeMiningAlgorithm)algorithm {
    if (!_isInitialized || !headerBytes) {
        return nil;
    }
    
    @try {
        std::vector<uint8_t> header(static_cast<const uint8_t*>(headerBytes.bytes), 
                                   static_cast<const uint8_t*>(headerBytes.bytes) + headerBytes.length);
        
        auto result = _miner->computeHash(header, static_cast<int>(algorithm));
        
        return [NSData dataWithBytes:result.data() length:result.size()];
        
    } @catch (NSException *exception) {
        NSLog(@"Error computing hash: %@", exception.reason);
        return nil;
    }
}

- (BOOL)validateThermalProof:(NSData *)proof temperature:(float)temperature {
    if (!_thermalManager || !proof) {
        return NO;
    }
    
    @try {
        std::vector<uint8_t> proofBytes(static_cast<const uint8_t*>(proof.bytes), 
                                       static_cast<const uint8_t*>(proof.bytes) + proof.length);
        
        return _thermalManager->validateProof(proofBytes, temperature);
        
    } @catch (NSException *exception) {
        NSLog(@"Error validating thermal proof: %@", exception.reason);
        return NO;
    }
}

#pragma mark - Private Methods

- (void)startStatsTimer {
    [self stopStatsTimer];
    
    _statsTimer = [NSTimer scheduledTimerWithTimeInterval:1.0 
                                                   target:self 
                                                 selector:@selector(updateStats) 
                                                 userInfo:nil 
                                                  repeats:YES];
}

- (void)stopStatsTimer {
    if (_statsTimer) {
        [_statsTimer invalidate];
        _statsTimer = nil;
    }
}

- (void)updateStats {
    if (self.statsUpdateCallback) {
        NativeMiningStats *stats = [self getCurrentStats];
        self.statsUpdateCallback(stats);
    }
}

- (NativeMiningStats *)emptyStats {
    NativeMiningStats *stats = [[NativeMiningStats alloc] init];
    stats.totalHashRate = 0.0;
    stats.randomXHashRate = 0.0;
    stats.mobileXHashRate = 0.0;
    stats.sharesSubmitted = 0;
    stats.blocksFound = 0;
    stats.npuUtilization = 0.0;
    stats.currentIntensity = NativeMiningIntensityDisabled;
    stats.currentAlgorithm = NativeMiningAlgorithmMobileX;
    stats.timestamp = [NSDate date];
    return stats;
}

@end 