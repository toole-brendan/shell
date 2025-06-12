#pragma once

#include <vector>
#include <memory>
#include <cstdint>

// Forward declaration for MLModel to avoid importing Core ML in header
#ifdef __OBJC__
@class MLModel;
@class MLMultiArray;
#else
typedef void MLModel;
typedef void MLMultiArray;
#endif

namespace shell {
namespace mobile {
namespace ios {

/**
 * Core ML NPU Provider for iOS
 * Integrates with Apple's Neural Engine via Core ML framework
 */
class CoreMLNPUProvider {
public:
    CoreMLNPUProvider();
    ~CoreMLNPUProvider();

    // Initialization
    bool initialize();
    void shutdown();

    // Core ML model management
    bool loadModel(const std::string& modelPath);
    void setMLModel(void* mlModel); // MLModel* passed from Objective-C++
    bool isModelLoaded() const;

    // NPU availability
    bool isNeuralEngineAvailable() const;
    bool canUseNeuralEngine() const;

    // Neural operations for mining
    std::vector<uint8_t> processConvolution(const std::vector<uint8_t>& input);
    std::vector<float> runInference(const std::vector<float>& input);
    
    // Performance metrics
    float getNPUUtilization() const;
    uint64_t getTotalInferences() const;
    double getAverageInferenceTime() const;

    // Configuration
    void setMaxInferenceTime(double maxTime);
    void enablePerformanceProfile(bool enable);

private:
    // Model state
    void* mlModel_;          // MLModel*
    bool modelLoaded_;
    bool neuralEngineAvailable_;
    
    // Performance tracking
    std::atomic<uint64_t> totalInferences_;
    std::atomic<double> totalInferenceTime_;
    std::atomic<float> currentUtilization_;
    
    // Configuration
    double maxInferenceTime_;
    bool performanceProfileEnabled_;
    
    // Core ML integration methods
    bool checkNeuralEngineSupport();
    void* createInputMultiArray(const std::vector<float>& data);
    std::vector<float> extractOutputFromMultiArray(void* multiArray);
    
    // Convolution operations for MobileX
    std::vector<uint8_t> depthwiseSeparableConvolution(const std::vector<uint8_t>& input);
    std::vector<uint8_t> applyActivationFunction(const std::vector<uint8_t>& input);
    
    // Performance monitoring
    void updateUtilizationMetrics();
    double measureInferenceTime(const std::function<void()>& operation);
    
    // Device detection
    bool isA17ProOrLater() const;
    bool isM1OrLater() const;
    std::string getDeviceModel() const;
    
    // Error handling
    void logCoreMLError(const std::string& operation, int errorCode);
};

// Utility functions for Core ML integration
namespace coreml_utils {
    // Convert between C++ and Core ML data formats
    std::vector<float> uint8ToFloat(const std::vector<uint8_t>& input);
    std::vector<uint8_t> floatToUint8(const std::vector<float>& input);
    
    // Tensor operations
    std::vector<float> reshapeTensor(const std::vector<float>& input, 
                                   const std::vector<int>& oldShape,
                                   const std::vector<int>& newShape);
    
    // Model validation
    bool validateModelCompatibility(void* mlModel);
    
    // Performance utilities
    bool shouldUseCPUFallback(double inferenceTime, float targetUtilization);
}

} // namespace ios
} // namespace mobile
} // namespace shell 