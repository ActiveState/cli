function f() { echo received SIGINT; }
trap f SIGINT
trap -p SIGINT
echo 'Start of script'
sleep 10000          
echo 'After first sleep or interrupt'
trap 'exit 123' SIGINT
sleep 2
echo 'After second sleep, but not printed after second interrupt'