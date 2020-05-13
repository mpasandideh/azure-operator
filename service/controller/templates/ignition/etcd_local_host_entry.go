package ignition

const EtcdLocalHostsEntry = `[Unit]
Description=Adds hosts file entry for etcd local name
After=k8s-setup-network-env.service
Wants=k8s-setup-network-env.service
[Service]
Type=oneshot
RemainAfterExit=yes
EnvironmentFile=/etc/network-environment
ExecStart=/bin/sh -c '\
grep "%H.08d6v.k8s.godsmack.westeurope.azure.gigantic.io" /etc/hosts || echo "${DEFAULT_IPV4}    %H.08d6v.k8s.godsmack.westeurope.azure.gigantic.io" >> /etc/hosts'
[Install]
WantedBy=multi-user.target
`
