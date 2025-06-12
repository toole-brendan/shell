#pragma once

#include <cstdint>
#include <vector>
#include <memory>
#include <functional>
#include <string>

#ifdef __ANDROID__
#include <android/NeuralNetworks.h>
#endif

namespace shell {
namespace mobile {

/**
 * Tensor data structure for NPU operations
 */
struct Tensor {
    std::vector<float> data;
    std::vector<int> shape;
    
    Tensor() = default;
    Tensor(const std::vector<float>& data, const std::vector<int>& shape) 
        : data(data), shape(shape) {}
    
    size_t size() const;
    bool isValid() const;
};

/**
 * NPU performance metrics
 */
struct NPUMetrics {
    float utilization;      // Percentage 0-100
    float powerUsage;       // Estimated power in watts
    uint64_t operations;    // Total operations performed
    double averageLatency;  // Average operation latency in ms
    
    NPUMetrics();
};

/**
 * Abstract NPU adapter interface
 * Provides platform-specific NPU access
 */
class NPUAdapter {
public:
    virtual ~NPUAdapter() = default;

    // Lifecycle
    virtual bool initialize() = 0;
    virtual void shutdown() = 0;
    virtual bool isAvailable() const = 0;

    // Capabilities
    virtual std::string getPlatformName() const = 0;
    virtual std::vector<uint8_t> getHardwareFingerprint() const = 0;
    virtual bool supportsTrustedExecution() const = 0;

    // Operations
    virtual bool executeConvolution(const Tensor& input, Tensor& output) = 0;
    virtual bool executeDepthwiseConvolution(const Tensor& input, Tensor& output) = 0;
    
    // Performance
    virtual NPUMetrics getMetrics() const = 0;
    virtual void resetMetrics() = 0;
};

/**
 * Android NNAPI adapter implementation
 */
#ifdef __ANDROID__
class AndroidNNAPIAdapter : public NPUAdapter {
public:
    AndroidNNAPIAdapter();
    ~AndroidNNAPIAdapter() override;

    // NPUAdapter interface
    bool initialize() override;
    void shutdown() override;
    bool isAvailable() const override;

    std::string getPlatformName() const override;
    std::vector<uint8_t> getHardwareFingerprint() const override;
    bool supportsTrustedExecution() const override;

    bool executeConvolution(const Tensor& input, Tensor& output) override;
    bool executeDepthwiseConvolution(const Tensor& input, Tensor& output) override;

    NPUMetrics getMetrics() const override;
    void resetMetrics() override;

private:
    ANeuralNetworksModel* model_;
    ANeuralNetworksCompilation* compilation_;
    ANeuralNetworksExecution* execution_;
    
    bool modelCreated_;
    bool compilationReady_;
    mutable NPUMetrics metrics_;
    
    bool createModel();
    bool compileModel();
    void updateMetrics(double latency);
};
#endif

/**
 * CPU fallback implementation for NPU operations
 */
class CPUNeuralFallback : public NPUAdapter {
public:
    CPUNeuralFallback();
    ~CPUNeuralFallback() override;

    // NPUAdapter interface
    bool initialize() override;
    void shutdown() override;
    bool isAvailable() const override { return true; }

    std::string getPlatformName() const override { return "CPU_Fallback"; }
    std::vector<uint8_t> getHardwareFingerprint() const override;
    bool supportsTrustedExecution() const override { return false; }

    bool executeConvolution(const Tensor& input, Tensor& output) override;
    bool executeDepthwiseConvolution(const Tensor& input, Tensor& output) override;

    NPUMetrics getMetrics() const override;
    void resetMetrics() override;

private:
    mutable NPUMetrics metrics_;
    
    void softwareConvolution(const Tensor& input, Tensor& output);
    void updateMetrics(double latency);
};

/**
 * NPU Integration Manager
 * Manages NPU operations and fallback to CPU
 */
class NPUIntegration {
public:
    NPUIntegration();
    ~NPUIntegration();

    // Initialization
    bool initializeNNAPI();
    bool initializeCPUFallback();
    void shutdown();

    // Status
    bool isNPUAvailable() const;
    bool isUsingFallback() const;
    std::string getCurrentAdapter() const;

    // Operations
    bool executeConvolution(const Tensor& input, Tensor& output);
    bool runConvolution(const std::vector<float>& input, std::vector<float>& output);
    
    // Neural mining operations
    bool processNeuralStep(const std::vector<uint8_t>& vmState, std::vector<uint8_t>& result);
    Tensor stateToTensor(const std::vector<uint8_t>& state);
    std::vector<uint8_t> tensorToState(const Tensor& tensor);

    // Performance monitoring
    NPUMetrics getAverageMetrics() const;
    float getUtilization() const;
    void resetMetrics();

    // Configuration
    void setFallbackFunction(std::function<bool(const Tensor&, Tensor&)> fallback);
    void enableFallback(bool enable);

private:
    std::unique_ptr<NPUAdapter> primaryAdapter_;
    std::unique_ptr<CPUNeuralFallback> fallbackAdapter_;
    
    bool npuAvailable_;
    bool fallbackEnabled_;
    bool usesFallback_;
    
    std::function<bool(const Tensor&, Tensor&)> customFallback_;
    
    // Model weights for convolution (lightweight)
    std::vector<float> convolutionWeights_;
    std::vector<float> depthwiseWeights_;
    
    // Performance tracking
    mutable NPUMetrics aggregateMetrics_;
    mutable std::chrono::time_point<std::chrono::steady_clock> lastMetricUpdate_;
    
    void initializeWeights();
    void updateAggregateMetrics(const NPUMetrics& metrics);
    
    // Platform detection
    std::unique_ptr<NPUAdapter> detectBestAdapter();
};

// Utility functions
Tensor createTensor(const std::vector<float>& data, const std::vector<int>& shape);
std::vector<float> flattenTensor(const Tensor& tensor);
bool validateTensorShape(const Tensor& tensor, const std::vector<int>& expectedShape);

} // namespace mobile
} // namespace shell 