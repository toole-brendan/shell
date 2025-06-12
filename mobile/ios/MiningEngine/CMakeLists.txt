cmake_minimum_required(VERSION 3.16.0)

project(ShellMiningEngine VERSION 1.0.0 LANGUAGES CXX OBJCXX C)

# iOS-specific configuration
set(CMAKE_OSX_DEPLOYMENT_TARGET "15.0")
set(CMAKE_SYSTEM_NAME iOS)
set(CMAKE_OSX_ARCHITECTURES arm64)

# C++ Standard
set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

# Optimization flags for ARM64
set(CMAKE_CXX_FLAGS_RELEASE "-O3 -DNDEBUG -ffast-math -march=armv8-a+crc+crypto")
set(CMAKE_CXX_FLAGS_DEBUG "-O0 -g -DDEBUG")

# Apple Silicon optimizations
if(CMAKE_OSX_ARCHITECTURES MATCHES "arm64")
    set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -mcpu=apple-a14")
    # Enable NEON and other ARM64 features
    set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -mfpu=neon -mfloat-abi=hard")
endif()

# Include directories
include_directories(
    ${CMAKE_CURRENT_SOURCE_DIR}
    ${CMAKE_CURRENT_SOURCE_DIR}/third_party/randomx/src
    ${CMAKE_CURRENT_SOURCE_DIR}/shared
)

# Find required frameworks
find_library(FOUNDATION_FRAMEWORK Foundation REQUIRED)
find_library(COREML_FRAMEWORK CoreML REQUIRED)
find_library(IOKIT_FRAMEWORK IOKit REQUIRED)
find_library(UIKIT_FRAMEWORK UIKit REQUIRED)
find_library(SECURITY_FRAMEWORK Security REQUIRED)
find_library(ACCELERATE_FRAMEWORK Accelerate REQUIRED)

# RandomX library (will need to be built separately)
# For now, we'll link against a pre-built version
find_library(RANDOMX_LIBRARY randomx PATHS ${CMAKE_CURRENT_SOURCE_DIR}/lib)

# Source files
set(MINING_ENGINE_SOURCES
    shell_mining_bridge.mm
    ios_mobile_randomx.cpp
    core_ml_npu_provider.cpp
    ios_thermal_manager.cpp
)

# Headers
set(MINING_ENGINE_HEADERS
    shell_mining_bridge.h
    ios_mobile_randomx.h
    core_ml_npu_provider.h
    ios_thermal_manager.h
)

# Create the mining engine library
add_library(ShellMiningEngine STATIC
    ${MINING_ENGINE_SOURCES}
    ${MINING_ENGINE_HEADERS}
)

# Compiler-specific flags
target_compile_options(ShellMiningEngine PRIVATE
    -Wall
    -Wextra
    -Wpedantic
    -Wno-unused-parameter
    -fno-exceptions
    -fno-rtti
    $<$<CONFIG:Release>:-flto>
)

# Objective-C++ specific flags
set_source_files_properties(shell_mining_bridge.mm PROPERTIES
    COMPILE_FLAGS "-fobjc-arc -fobjc-weak"
)

# Link frameworks
target_link_libraries(ShellMiningEngine
    ${FOUNDATION_FRAMEWORK}
    ${COREML_FRAMEWORK}
    ${IOKIT_FRAMEWORK}
    ${UIKIT_FRAMEWORK}
    ${SECURITY_FRAMEWORK}
    ${ACCELERATE_FRAMEWORK}
)

# Link RandomX if available
if(RANDOMX_LIBRARY)
    target_link_libraries(ShellMiningEngine ${RANDOMX_LIBRARY})
else()
    message(WARNING "RandomX library not found, building stub implementation")
    target_compile_definitions(ShellMiningEngine PRIVATE RANDOMX_STUB=1)
endif()

# Platform-specific definitions
target_compile_definitions(ShellMiningEngine PRIVATE
    SHELL_MOBILE_IOS=1
    APPLE_SILICON_OPTIMIZED=1
)

# iOS-specific compile definitions
if(CMAKE_OSX_DEPLOYMENT_TARGET VERSION_GREATER_EQUAL "15.0")
    target_compile_definitions(ShellMiningEngine PRIVATE IOS_15_PLUS=1)
endif()

# Core ML availability check
if(CMAKE_OSX_DEPLOYMENT_TARGET VERSION_GREATER_EQUAL "14.0")
    target_compile_definitions(ShellMiningEngine PRIVATE COREML_AVAILABLE=1)
endif()

# Debug configuration
if(CMAKE_BUILD_TYPE STREQUAL "Debug")
    target_compile_definitions(ShellMiningEngine PRIVATE
        SHELL_DEBUG=1
        ENABLE_LOGGING=1
    )
endif()

# Installation
install(TARGETS ShellMiningEngine
    ARCHIVE DESTINATION lib
    LIBRARY DESTINATION lib
    FRAMEWORK DESTINATION Frameworks
)

install(FILES ${MINING_ENGINE_HEADERS}
    DESTINATION include/shell
)

# Xcode-specific settings
if(CMAKE_GENERATOR STREQUAL "Xcode")
    set_target_properties(ShellMiningEngine PROPERTIES
        XCODE_ATTRIBUTE_DEVELOPMENT_TEAM "${IOS_DEVELOPMENT_TEAM}"
        XCODE_ATTRIBUTE_CODE_SIGN_IDENTITY "iPhone Developer"
        XCODE_ATTRIBUTE_PROVISIONING_PROFILE_SPECIFIER "${IOS_PROVISIONING_PROFILE}"
        XCODE_ATTRIBUTE_PRODUCT_BUNDLE_IDENTIFIER "com.shell.mining.engine"
        XCODE_ATTRIBUTE_IPHONEOS_DEPLOYMENT_TARGET ${CMAKE_OSX_DEPLOYMENT_TARGET}
        XCODE_ATTRIBUTE_ARCHS "arm64"
        XCODE_ATTRIBUTE_VALID_ARCHS "arm64"
        XCODE_ATTRIBUTE_ENABLE_BITCODE "NO"
        XCODE_ATTRIBUTE_CLANG_ENABLE_OBJC_ARC "YES"
        XCODE_ATTRIBUTE_CLANG_ENABLE_OBJC_WEAK "YES"
        XCODE_ATTRIBUTE_GCC_C_LANGUAGE_STANDARD "c11"
        XCODE_ATTRIBUTE_CLANG_CXX_LANGUAGE_STANDARD "c++17"
        XCODE_ATTRIBUTE_CLANG_CXX_LIBRARY "libc++"
    )
endif()

# Performance optimization targets
add_custom_target(optimize
    COMMAND ${CMAKE_COMMAND} -E echo "Applying iOS-specific optimizations..."
    COMMAND ${CMAKE_COMMAND} -DCMAKE_BUILD_TYPE=Release ${CMAKE_SOURCE_DIR}
    WORKING_DIRECTORY ${CMAKE_BINARY_DIR}
)

# Test target (if tests are available)
if(EXISTS ${CMAKE_CURRENT_SOURCE_DIR}/tests)
    add_subdirectory(tests)
endif()

# Documentation
add_custom_target(docs
    COMMAND doxygen ${CMAKE_CURRENT_SOURCE_DIR}/Doxyfile
    WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}
    COMMENT "Generating documentation with Doxygen"
    VERBATIM
)

# Print configuration summary
message(STATUS "")
message(STATUS "Shell Mining Engine iOS Configuration Summary:")
message(STATUS "  Target System: ${CMAKE_SYSTEM_NAME}")
message(STATUS "  Architecture: ${CMAKE_OSX_ARCHITECTURES}")
message(STATUS "  Deployment Target: ${CMAKE_OSX_DEPLOYMENT_TARGET}")
message(STATUS "  Build Type: ${CMAKE_BUILD_TYPE}")
message(STATUS "  C++ Standard: ${CMAKE_CXX_STANDARD}")
message(STATUS "  RandomX Library: ${RANDOMX_LIBRARY}")
message(STATUS "") 