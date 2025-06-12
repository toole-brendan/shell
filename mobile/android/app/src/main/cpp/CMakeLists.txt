cmake_minimum_required(VERSION 3.22.1)

project("shellmining")

# Enable C++17
set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# Optimization flags for ARM64
set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} -O3 -ffast-math")
set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -O3 -ffast-math")

# ARM64 specific optimizations
if(${ANDROID_ABI} STREQUAL "arm64-v8a")
    set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} -march=armv8-a+simd")
    set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -march=armv8-a+simd")
    add_definitions(-DARM64_OPTIMIZATIONS=1)
    add_definitions(-DNEON_SUPPORT=1)
endif()

# Find required Android libraries
find_library(log-lib log)
find_library(android-lib android)

# Neural Networks API (available on API 27+)
find_library(neuralnetworks-lib neuralnetworks)

# Include directories
include_directories(
    .
    ../../../shared/mining-core
    ../../../shared/protocols
    ../../../shared/crypto
    ${CMAKE_CURRENT_SOURCE_DIR}/../../../../mining/mobilex
    ${CMAKE_CURRENT_SOURCE_DIR}/../../../../third_party/randomx/src
)

# Source files
set(SHELL_MINING_SOURCES
    shell_mining_jni.cpp
    mobile_randomx.cpp
    thermal_verification.cpp
    arm64_optimizations.cpp
    npu_integration.cpp
    android_power_manager.cpp
    android_thermal_manager.cpp
)

# Shared C++ sources (to be implemented in future phases)
# set(SHARED_SOURCES
#     ../../../shared/mining-core/mobilex_core.cpp
#     ../../../shared/mining-core/randomx_wrapper.cpp
#     ../../../shared/crypto/hash_functions.cpp
#     ../../../shared/protocols/stratum_client.cpp
# )

# RandomX sources (subset needed for mobile)
set(RANDOMX_SOURCES
    ../../../../third_party/randomx/src/common.cpp
    ../../../../third_party/randomx/src/aes_hash.cpp
    ../../../../third_party/randomx/src/argon2_core.cpp
    ../../../../third_party/randomx/src/argon2_ref.cpp
    ../../../../third_party/randomx/src/blake2_generator.cpp
    ../../../../third_party/randomx/src/blake2b.c
    ../../../../third_party/randomx/src/bytecode_machine.cpp
    ../../../../third_party/randomx/src/cpu.cpp
    ../../../../third_party/randomx/src/dataset.cpp
    ../../../../third_party/randomx/src/soft_aes.cpp
    ../../../../third_party/randomx/src/virtual_machine.cpp
    ../../../../third_party/randomx/src/virtual_memory.cpp
    ../../../../third_party/randomx/src/vm_interpreted.cpp
    ../../../../third_party/randomx/src/vm_interpreted_light.cpp
    ../../../../third_party/randomx/src/vm_compiled.cpp
    ../../../../third_party/randomx/src/vm_compiled_light.cpp
    ../../../../third_party/randomx/src/allocator.cpp
    ../../../../third_party/randomx/src/assembly_generator_x86.cpp
    ../../../../third_party/randomx/src/instruction.cpp
    ../../../../third_party/randomx/src/randomx.cpp
    ../../../../third_party/randomx/src/superscalar.cpp
    ../../../../third_party/randomx/src/program.cpp
    ../../../../third_party/randomx/src/reciprocal.c
    ../../../../third_party/randomx/src/configuration.cpp
)

# ARM64 specific RandomX sources
if(${ANDROID_ABI} STREQUAL "arm64-v8a")
    list(APPEND RANDOMX_SOURCES
        ../../../../third_party/randomx/src/jit_compiler_a64.cpp
        ../../../../third_party/randomx/src/jit_compiler_a64_static.S
    )
endif()

# Create the shared library
add_library(shellmining SHARED
    ${SHELL_MINING_SOURCES}
    # ${SHARED_SOURCES}  # Will be added in future phases
    ${RANDOMX_SOURCES}
)

# Preprocessor definitions
target_compile_definitions(shellmining PRIVATE
    RANDOMX_ARM=1
    RANDOMX_JIT=1
    ANDROID=1
    __ANDROID__=1
)

# Link libraries
target_link_libraries(shellmining
    ${log-lib}
    ${android-lib}
    ${neuralnetworks-lib}
    crypto   # OpenSSL crypto library for SHA256
    ssl      # OpenSSL SSL library
)

# Enable NEON optimizations if available
if(${ANDROID_ABI} STREQUAL "arm64-v8a")
    target_compile_options(shellmining PRIVATE -mfpu=neon)
endif()

# Link time optimizations for release builds
if(CMAKE_BUILD_TYPE STREQUAL "Release")
    set_target_properties(shellmining PROPERTIES
        LINK_FLAGS "-Wl,--gc-sections -flto"
    )
endif()

# Strip debug symbols in release builds
if(CMAKE_BUILD_TYPE STREQUAL "Release")
    add_custom_command(TARGET shellmining POST_BUILD
        COMMAND ${CMAKE_STRIP} $<TARGET_FILE:shellmining>
        COMMENT "Stripping debug symbols"
    )
endif() 