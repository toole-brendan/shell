#pragma once

#import <Foundation/Foundation.h>
#import <CoreML/CoreML.h>

NS_ASSUME_NONNULL_BEGIN

// Mining intensity levels matching Swift enum
typedef NS_ENUM(NSInteger, NativeMiningIntensity) {
    NativeMiningIntensityDisabled = 0,
    NativeMiningIntensityLight = 1,
    NativeMiningIntensityMedium = 2,
    NativeMiningIntensityFull = 3
};

// Mining algorithm types
typedef NS_ENUM(NSInteger, NativeMiningAlgorithm) {
    NativeMiningAlgorithmRandomX = 0,
    NativeMiningAlgorithmMobileX = 1,
    NativeMiningAlgorithmDual = 2
};

// Mining statistics structure
@interface NativeMiningStats : NSObject
@property (nonatomic, assign) double totalHashRate;
@property (nonatomic, assign) double randomXHashRate;
@property (nonatomic, assign) double mobileXHashRate;
@property (nonatomic, assign) int64_t sharesSubmitted;
@property (nonatomic, assign) int32_t blocksFound;
@property (nonatomic, assign) float npuUtilization;
@property (nonatomic, assign) NativeMiningIntensity currentIntensity;
@property (nonatomic, assign) NativeMiningAlgorithm currentAlgorithm;
@property (nonatomic, strong) NSDate *timestamp;
@end

// Mining configuration structure
@interface NativeMiningConfig : NSObject
@property (nonatomic, assign) NativeMiningIntensity intensity;
@property (nonatomic, assign) NativeMiningAlgorithm algorithm;
@property (nonatomic, assign) BOOL npuEnabled;
@property (nonatomic, assign) float maxTemperature;
@property (nonatomic, assign) float throttleTemperature;
@property (nonatomic, strong) NSString *poolAddress;
@property (nonatomic, assign) NSUInteger coreCount;
@end

// Power state structure
@interface NativePowerState : NSObject
@property (nonatomic, assign) int32_t batteryLevel;
@property (nonatomic, assign) BOOL isCharging;
@property (nonatomic, assign) BOOL isPowerSaveMode;
@property (nonatomic, assign) NSInteger thermalState;
@property (nonatomic, strong) NSDate *timestamp;
@end

// Thermal state structure
@interface NativeThermalState : NSObject
@property (nonatomic, assign) float temperature;
@property (nonatomic, assign) NSInteger state;
@property (nonatomic, assign) BOOL isThrottling;
@property (nonatomic, strong) NSDate *timestamp;
@end

// Main mining bridge interface
@interface ShellMiningBridge : NSObject

// Initialization
- (instancetype)init;
- (void)dealloc;

// Mining lifecycle
- (BOOL)initializeWithConfig:(NativeMiningConfig *)config error:(NSError **)error;
- (BOOL)startMining:(NSError **)error;
- (BOOL)stopMining:(NSError **)error;
- (void)shutdown;

// Configuration
- (BOOL)updateConfig:(NativeMiningConfig *)config error:(NSError **)error;
- (void)configureNPUWithModel:(MLModel * _Nullable)model;

// Status and metrics
- (NativeMiningStats *)getCurrentStats;
- (BOOL)isMining;
- (uint64_t)getTotalHashes;

// Power and thermal management
- (NativePowerState *)getCurrentPowerState;
- (NativeThermalState *)getCurrentThermalState;
- (BOOL)canMineAtIntensity:(NativeMiningIntensity)intensity;

// Mining operations
- (NSData * _Nullable)computeHashForHeader:(NSData *)headerBytes algorithm:(NativeMiningAlgorithm)algorithm;
- (BOOL)validateThermalProof:(NSData *)proof temperature:(float)temperature;

// Callbacks for async operations
@property (nonatomic, copy, nullable) void (^statsUpdateCallback)(NativeMiningStats *stats);
@property (nonatomic, copy, nullable) void (^shareFoundCallback)(NSData *share, double difficulty);
@property (nonatomic, copy, nullable) void (^errorCallback)(NSError *error);

@end

NS_ASSUME_NONNULL_END 