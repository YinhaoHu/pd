function main {
    local HOST_IP="127.0.0.1"
    local PORT_CLIENT=32379
    local PORT_PEER=32380
    ../bin/pd-server --name="pd-testing" \
        --data-dir="user-local/pd-testing" \
        --client-urls="http://${HOST_IP}:${PORT_CLIENT}" \
        --peer-urls="http://${HOST_IP}:${PORT_PEER}" \
        --log-file=/Users/huyinhao/filebase/project/tikv-projects/pd/testing-client/user-local/pd.log
        --log-level="debug"
}

main
