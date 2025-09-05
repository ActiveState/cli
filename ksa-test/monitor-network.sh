#!/bin/bash

# Network monitoring script for state-installer testing
# This helps inspect network traffic during installation

echo "=== Network Monitoring for State Tool Installer ==="
echo ""

# Function to show current network connections
show_connections() {
    echo "=== Current Network Connections ==="
    netstat -tuln 2>/dev/null || ss -tuln 2>/dev/null || echo "netstat/ss not available"
    echo ""
}

# Function to show processes using network
show_network_processes() {
    echo "=== Processes Using Network ==="
    lsof -i 2>/dev/null || echo "lsof not available"
    echo ""
}

# Function to start packet capture (if tcpdump is available)
start_packet_capture() {
    if command -v tcpdump >/dev/null 2>&1; then
        echo "=== Starting Packet Capture ==="
        echo "Capturing packets to /tmp/state-installer-capture.pcap"
        echo "Press Ctrl+C to stop capture"
        echo ""
        tcpdump -i any -w /tmp/state-installer-capture.pcap host platform.activestate.com or host state-tool.s3.amazonaws.com &
        TCPDUMP_PID=$!
        echo "Packet capture started with PID: $TCPDUMP_PID"
        echo "To stop: kill $TCPDUMP_PID"
        echo ""
    else
        echo "tcpdump not available for packet capture"
    fi
}

# Function to monitor DNS lookups
monitor_dns() {
    echo "=== Monitoring DNS Lookups ==="
    echo "Watching for DNS queries to activestate.com and amazonaws.com"
    echo "Press Ctrl+C to stop monitoring"
    echo ""
    
    # Use strace to monitor DNS calls if available
    if command -v strace >/dev/null 2>&1; then
        strace -e trace=network -f -p $$ 2>&1 | grep -E "(connect|sendto|recvfrom)" &
        STRACE_PID=$!
        echo "DNS monitoring started with PID: $STRACE_PID"
        echo "To stop: kill $STRACE_PID"
    else
        echo "strace not available for DNS monitoring"
    fi
}

# Function to show network interfaces
show_interfaces() {
    echo "=== Network Interfaces ==="
    ip addr show 2>/dev/null || ifconfig 2>/dev/null || echo "Network interface tools not available"
    echo ""
}

# Function to test connectivity to State Tool endpoints
test_connectivity() {
    echo "=== Testing Connectivity to State Tool Endpoints ==="
    
    echo "Testing platform.activestate.com..."
    curl -I -s --connect-timeout 5 https://platform.activestate.com || echo "Failed to connect to platform.activestate.com"
    
    echo "Testing state-tool.s3.amazonaws.com..."
    curl -I -s --connect-timeout 5 https://state-tool.s3.amazonaws.com || echo "Failed to connect to state-tool.s3.amazonaws.com"
    
    echo ""
}

# Main menu
show_menu() {
    echo "Network Monitoring Options:"
    echo "1. Show current network connections"
    echo "2. Show processes using network"
    echo "3. Start packet capture"
    echo "4. Monitor DNS lookups"
    echo "5. Show network interfaces"
    echo "6. Test connectivity to State Tool endpoints"
    echo "7. Run all monitoring (except packet capture)"
    echo "8. Exit"
    echo ""
}

# Cleanup function
cleanup() {
    echo ""
    echo "Cleaning up monitoring processes..."
    if [ ! -z "$TCPDUMP_PID" ]; then
        kill $TCPDUMP_PID 2>/dev/null || true
        echo "Stopped packet capture"
    fi
    if [ ! -z "$STRACE_PID" ]; then
        kill $STRACE_PID 2>/dev/null || true
        echo "Stopped DNS monitoring"
    fi
}

# Set up signal handlers
trap cleanup EXIT INT TERM

# Main loop
while true; do
    show_menu
    read -p "Select option (1-8): " choice
    
    case $choice in
        1)
            show_connections
            ;;
        2)
            show_network_processes
            ;;
        3)
            start_packet_capture
            ;;
        4)
            monitor_dns
            ;;
        5)
            show_interfaces
            ;;
        6)
            test_connectivity
            ;;
        7)
            show_connections
            show_network_processes
            show_interfaces
            test_connectivity
            ;;
        8)
            echo "Exiting network monitor..."
            break
            ;;
        *)
            echo "Invalid option. Please select 1-8."
            ;;
    esac
    
    echo ""
    read -p "Press Enter to continue..."
    echo ""
done
