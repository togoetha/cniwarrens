#!/bin/bash -v

echo "--- Enabling NAT ---"

wget -O - -nv --ciphers DEFAULT@SECLEVEL=1 https://www.wall2.ilabt.iminds.be/enable-nat.sh | sudo bash

echo "--- Installing Containerd ---"
#modprobe overlay
#modprobe br_netfilter

apt-get update

apt-get install -y \
    ca-certificates \
    curl \
    gnupg \
    lsb-release

mkdir -p /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

apt-get update
apt-get install -y containerd.io cgroup-tools


echo "--- Installing xdp/ebpf libraries ---"
apt-get install -y clang llvm libelf-dev libpcap-dev gcc-multilib build-essential linux-tools-common linux-tools-generic tcpdump
apt-get install -y linux-tools-$(uname -r)
apt-get install -y linux-headers-$(uname -r)

#xdp tools
echo "--- Installing xdp-tools ---"
cd ..
git clone https://github.com/xdp-project/xdp-tools
cd xdp-tools
make
make install
cd ..

echo "--- Getting XDP tutorial for basic setup script ---"
#xdp tutorial setup scripts for testing, comment out if required
git clone https://github.com/xdp-project/xdp-tutorial

echo "--- Installing golang ---"
#golang for compiling, just in case
wget https://go.dev/dl/go1.19.4.linux-amd64.tar.gz
rm -rf /usr/local/go && tar -C /usr/local -xzf go1.19.4.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin