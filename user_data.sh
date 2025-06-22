#!/bin/bash
set -eux

# -------------------------------
# 1. Install packages
# -------------------------------
sudo apt update && sudo apt install -y wireguard

# -------------------------------
# 2. Enable IP forwarding
# -------------------------------
sudo tee -a /etc/sysctl.conf <<EOF
net.ipv4.ip_forward=1
net.ipv6.conf.all.forwarding=1
EOF
sudo sysctl -p

# -------------------------------
# 4. Write server WireGuard config (/etc/wireguard/wg0.conf)
# -------------------------------
sudo tee /etc/wireguard/wg0.conf <<EOF
%s
EOF

# Configure Networking

sudo ufw route allow in on wg0 out on eth0
sudo iptables -t nat -I POSTROUTING -o eth0 -j MASQUERADE
sudo ip6tables -t nat -I POSTROUTING -o eth0 -j MASQUERADE

# -------------------------------
# 6. Enable and start the WireGuard interface
# -------------------------------
sudo systemctl enable wg-quick@wg0
sudo systemctl start wg-quick@wg0
