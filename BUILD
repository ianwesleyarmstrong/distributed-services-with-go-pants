local_environment(name="local_osx", compatible_platforms=["macos_arm64"])

docker_environment(
    name="linux_docker",
    platform="linux_x86_64",
    image="ubuntu:22.04",
)

local_environment(
    name="local_osx_go",
    compatible_platforms=["macos_arm64"],
    fallback_environment="linux_go",
)
docker_environment(name="linux_go", platform="linux_x86_64", image="golang:1.20")
