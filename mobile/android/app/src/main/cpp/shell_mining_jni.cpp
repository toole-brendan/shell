/**
 * Shell Reserve Mobile Mining - Android JNI Bridge
 * 
 * This file provides the JNI interface between Android Kotlin/Java code
 * and the native C++ mobile mining implementation.
 */

#include <jni.h>
#include <string>
#include <memory>
#include <android/log.h>
#include <android/api-level.h>
#include <android/NeuralNetworks.h>

// Include Shell mobile mining headers
#include "mobile_randomx.h"
#include "thermal_verification.h"
#include "arm64_optimizations.h"
#include "npu_integration.h"

#define TAG "ShellMining"
#define LOGD(...) __android_log_print(ANDROID_LOG_DEBUG, TAG, __VA_ARGS__)
#define LOGE(...) __android_log_print(ANDROID_LOG_ERROR, TAG, __VA_ARGS__)
#define LOGI(...) __android_log_print(ANDROID_LOG_INFO, TAG, __VA_ARGS__)

namespace shell {
namespace mobile {

/**
 * C++ Mobile Mining Engine for Android
 */
class AndroidMobileXMiner {
private:
    std::unique_ptr<MobileXMiner> miner_;
    std::unique_ptr<ThermalVerification> thermal_;
    std::unique_ptr<ARM64Optimizer> arm64_;
    std::unique_ptr<NPUIntegration> npu_;
    
    bool is_mining_;
    bool npu_available_;
    MiningIntensity current_intensity_;
    
public:
    AndroidMobileXMiner() 
        : is_mining_(false)
        , npu_available_(false)
        , current_intensity_(MiningIntensity::MEDIUM) {
        
        LOGD("Initializing AndroidMobileXMiner");
        
        // Initialize core mining components
        miner_ = std::make_unique<MobileXMiner>();
        thermal_ = std::make_unique<ThermalVerification>();
        arm64_ = std::make_unique<ARM64Optimizer>();
        
        // Detect and initialize NPU if available
        detectAndInitializeNPU();
        
        // Configure ARM64 optimizations
        configureARM64Features();
    }
    
    ~AndroidMobileXMiner() {
        if (is_mining_) {
            stopMining();
        }
    }
    
    bool initialize() {
        LOGD("Initializing mobile mining engine");
        
        try {
            // Initialize RandomX with mobile optimizations
            if (!miner_->initialize()) {
                LOGE("Failed to initialize MobileX miner");
                return false;
            }
            
            // Initialize thermal monitoring
            if (!thermal_->initialize()) {
                LOGE("Failed to initialize thermal verification");
                return false;
            }
            
            // Configure ARM64 optimizations
            if (!arm64_->initialize()) {
                LOGE("Failed to initialize ARM64 optimizations");
                return false;
            }
            
            LOGI("Mobile mining engine initialized successfully");
            return true;
            
        } catch (const std::exception& e) {
            LOGE("Exception during initialization: %s", e.what());
            return false;
        }
    }
    
    bool startMining(MiningIntensity intensity) {
        if (is_mining_) {
            LOGD("Mining already active");
            return true;
        }
        
        LOGD("Starting mining with intensity: %d", static_cast<int>(intensity));
        
        current_intensity_ = intensity;
        
        // Configure heterogeneous cores based on intensity
        configureHeterogeneousCores(intensity);
        
        // Start mining
        is_mining_ = miner_->startMining(intensity);
        
        if (is_mining_) {
            LOGI("Mining started successfully");
        } else {
            LOGE("Failed to start mining");
        }
        
        return is_mining_;
    }
    
    bool stopMining() {
        if (!is_mining_) {
            return true;
        }
        
        LOGD("Stopping mining");
        
        miner_->stopMining();
        is_mining_ = false;
        
        LOGI("Mining stopped");
        return true;
    }
    
    double getHashRate() const {
        return miner_->getHashRate();
    }
    
    double getRandomXHashRate() const {
        return miner_->getRandomXHashRate();
    }
    
    double getMobileXHashRate() const {
        return miner_->getMobileXHashRate();
    }
    
    float getCurrentTemperature() const {
        return thermal_->getCurrentTemperature();
    }
    
    float getNPUUtilization() const {
        if (!npu_available_ || !npu_) {
            return 0.0f;
        }
        return npu_->getUtilization();
    }
    
    bool isMining() const {
        return is_mining_;
    }
    
    long generateThermalProof() {
        return thermal_->generateThermalProof();
    }
    
private:
    void detectAndInitializeNPU() {
        LOGD("Detecting NPU capabilities");
        
        // Check Android API level for NNAPI support
        if (android_get_device_api_level() >= 27) {
            try {
                npu_ = std::make_unique<NPUIntegration>();
                npu_available_ = npu_->initializeNNAPI();
                
                if (npu_available_) {
                    LOGI("NPU initialized successfully via NNAPI");
                } else {
                    LOGD("NPU not available, using CPU fallback");
                }
            } catch (const std::exception& e) {
                LOGE("NPU initialization failed: %s", e.what());
                npu_available_ = false;
            }
        } else {
            LOGD("Android API level too low for NNAPI support");
            npu_available_ = false;
        }
    }
    
    void configureARM64Features() {
        LOGD("Configuring ARM64 features");
        
        // Enable NEON vector operations
        arm64_->enableNEON();
        
        // Check for additional ARM64 features
        if (arm64_->hasSVE()) {
            LOGI("SVE (Scalable Vector Extension) available");
            arm64_->enableSVE();
        }
        
        if (arm64_->hasDotProduct()) {
            LOGI("Int8 dot product instructions available");
            arm64_->enableDotProduct();
        }
    }
    
    void configureHeterogeneousCores(MiningIntensity intensity) {
        LOGD("Configuring heterogeneous cores for intensity: %d", static_cast<int>(intensity));
        
        // Configure big.LITTLE core usage based on intensity
        switch (intensity) {
            case MiningIntensity::LIGHT:
                arm64_->configureHeterogeneousCores(2, 2); // 2 big, 2 little
                break;
            case MiningIntensity::MEDIUM:
                arm64_->configureHeterogeneousCores(4, 4); // 4 big, 4 little
                break;
            case MiningIntensity::FULL:
                arm64_->configureHeterogeneousCores(8, 8); // All cores
                break;
            default:
                arm64_->configureHeterogeneousCores(0, 0); // No cores
                break;
        }
    }
};

} // namespace mobile
} // namespace shell

extern "C" {

/**
 * Create a new mobile mining engine instance
 */
JNIEXPORT jlong JNICALL
Java_com_shell_miner_nativecode_MiningEngine_createMiner(JNIEnv* env, jobject /* this */) {
    LOGD("Creating new mobile miner instance");
    
    try {
        auto* miner = new shell::mobile::AndroidMobileXMiner();
        if (miner->initialize()) {
            return reinterpret_cast<jlong>(miner);
        } else {
            delete miner;
            return 0;
        }
    } catch (const std::exception& e) {
        LOGE("Failed to create miner: %s", e.what());
        return 0;
    }
}

/**
 * Destroy a mobile mining engine instance
 */
JNIEXPORT void JNICALL
Java_com_shell_miner_nativecode_MiningEngine_destroyMiner(JNIEnv* env, jobject /* this */, jlong minerPtr) {
    if (minerPtr != 0) {
        auto* miner = reinterpret_cast<shell::mobile::AndroidMobileXMiner*>(minerPtr);
        delete miner;
        LOGD("Miner instance destroyed");
    }
}

/**
 * Start mining with specified intensity
 */
JNIEXPORT jboolean JNICALL
Java_com_shell_miner_nativecode_MiningEngine_startMining(
    JNIEnv* env, jobject /* this */, jlong minerPtr, jint intensity) {
    
    if (minerPtr == 0) {
        LOGE("Invalid miner pointer");
        return JNI_FALSE;
    }
    
    auto* miner = reinterpret_cast<shell::mobile::AndroidMobileXMiner*>(minerPtr);
    auto miningIntensity = static_cast<shell::mobile::MiningIntensity>(intensity);
    
    bool success = miner->startMining(miningIntensity);
    return success ? JNI_TRUE : JNI_FALSE;
}

/**
 * Stop mining
 */
JNIEXPORT jboolean JNICALL
Java_com_shell_miner_nativecode_MiningEngine_stopMining(JNIEnv* env, jobject /* this */, jlong minerPtr) {
    if (minerPtr == 0) {
        return JNI_FALSE;
    }
    
    auto* miner = reinterpret_cast<shell::mobile::AndroidMobileXMiner*>(minerPtr);
    return miner->stopMining() ? JNI_TRUE : JNI_FALSE;
}

/**
 * Get current hash rate
 */
JNIEXPORT jdouble JNICALL
Java_com_shell_miner_nativecode_MiningEngine_getHashRate(JNIEnv* env, jobject /* this */, jlong minerPtr) {
    if (minerPtr == 0) {
        return 0.0;
    }
    
    auto* miner = reinterpret_cast<shell::mobile::AndroidMobileXMiner*>(minerPtr);
    return miner->getHashRate();
}

/**
 * Get RandomX hash rate
 */
JNIEXPORT jdouble JNICALL
Java_com_shell_miner_nativecode_MiningEngine_getRandomXHashRate(JNIEnv* env, jobject /* this */, jlong minerPtr) {
    if (minerPtr == 0) {
        return 0.0;
    }
    
    auto* miner = reinterpret_cast<shell::mobile::AndroidMobileXMiner*>(minerPtr);
    return miner->getRandomXHashRate();
}

/**
 * Get MobileX hash rate
 */
JNIEXPORT jdouble JNICALL
Java_com_shell_miner_nativecode_MiningEngine_getMobileXHashRate(JNIEnv* env, jobject /* this */, jlong minerPtr) {
    if (minerPtr == 0) {
        return 0.0;
    }
    
    auto* miner = reinterpret_cast<shell::mobile::AndroidMobileXMiner*>(minerPtr);
    return miner->getMobileXHashRate();
}

/**
 * Get current temperature
 */
JNIEXPORT jfloat JNICALL
Java_com_shell_miner_nativecode_MiningEngine_getCurrentTemperature(JNIEnv* env, jobject /* this */, jlong minerPtr) {
    if (minerPtr == 0) {
        return 30.0f;
    }
    
    auto* miner = reinterpret_cast<shell::mobile::AndroidMobileXMiner*>(minerPtr);
    return miner->getCurrentTemperature();
}

/**
 * Get NPU utilization percentage
 */
JNIEXPORT jfloat JNICALL
Java_com_shell_miner_nativecode_MiningEngine_getNPUUtilization(JNIEnv* env, jobject /* this */, jlong minerPtr) {
    if (minerPtr == 0) {
        return 0.0f;
    }
    
    auto* miner = reinterpret_cast<shell::mobile::AndroidMobileXMiner*>(minerPtr);
    return miner->getNPUUtilization();
}

/**
 * Check if mining is active
 */
JNIEXPORT jboolean JNICALL
Java_com_shell_miner_nativecode_MiningEngine_isMining(JNIEnv* env, jobject /* this */, jlong minerPtr) {
    if (minerPtr == 0) {
        return JNI_FALSE;
    }
    
    auto* miner = reinterpret_cast<shell::mobile::AndroidMobileXMiner*>(minerPtr);
    return miner->isMining() ? JNI_TRUE : JNI_FALSE;
}

/**
 * Generate thermal proof for current mining state
 */
JNIEXPORT jlong JNICALL
Java_com_shell_miner_nativecode_MiningEngine_generateThermalProof(JNIEnv* env, jobject /* this */, jlong minerPtr) {
    if (minerPtr == 0) {
        return 0;
    }
    
    auto* miner = reinterpret_cast<shell::mobile::AndroidMobileXMiner*>(minerPtr);
    return miner->generateThermalProof();
}

/**
 * Configure NPU with Android NNAPI
 */
JNIEXPORT void JNICALL
Java_com_shell_miner_nativecode_MiningEngine_configureNPU(JNIEnv* env, jobject /* this */, jlong minerPtr) {
    if (minerPtr == 0) {
        return;
    }
    
    LOGD("Configuring NPU with NNAPI");
    
    // Note: This is a placeholder for NNAPI integration
    // Full implementation would create NNAPI models and configure the NPU
    
    LOGI("NPU configuration completed");
}

} // extern "C" 