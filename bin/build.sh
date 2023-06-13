#!/usr/bin/env bash

# Argument #1: Root path of project, current directory assumed when not defined
root_path=$(readlink -f "${1:-$PWD}")

# Argument #2: Go program path to build, "internal/camscan/camscan.go" assumed when not defined
build_path="${2:-internal/camscan/camscan.go}"

# Argument #3: The platforms and architectures to build binaries for
builds="${3:-aix=ppc64|android=386,amd64,arm,arm64|darwin=amd64,arm64|dragonfly=amd64|freebsd=386,amd64,arm,arm64,riscv64|illumos=amd64|ios=amd64,arm64|js=wasm|linux=386,amd64,arm,arm64,loong64,mips,mips64,mips64le,mipsle,ppc64,ppc64le,riscv64,s390x|netbsd=386,amd64,arm,arm64|openbsd=386,amd64,arm,arm64,mips64|plan9=386,amd64,arm|solaris=amd64|windows=386,amd64,arm,arm64}"

# Path to the bin directory where the final binaries should be placed
bin_path=$(readlink -f "$root_path/bin")

printf "\n Building CamScan program: %s\n\n" "$build_path"

IFS="|" read -r -a platforms <<< "$builds"

for platform_conf in "${platforms[@]}"
do
   :
   IFS="=" read -r -a parts <<< "$platform_conf"
   platform="${parts[0]}"
   arch_spec="${parts[1]}"
   IFS="," read -r -a architectures <<< "$arch_spec"
   printf " ∟₋ Building Platform: %s\n" "$platform"
   for architecture in "${architectures[@]}"
   do
     :
     printf "  ∟₋₋ Building Architecture: %s\n" "$architecture"
     output_path="$bin_path/camscan-$platform-$architecture"
     if [ "$platform" == "windows" ]; then
       output_path="$output_path.exe"
     fi
     export GOOS="$platform"
     export GOARCH="$architecture"
     go build -trimpath -o "$output_path" "$build_path"
     printf "  ∟₋₋ Built %s to %s\n" "$architecture" "$output_path"
   done
   printf "\n"
done
