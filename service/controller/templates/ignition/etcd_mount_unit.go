package ignition

const EtcdMountUnit = `[Unit]
Description=Mounts azure file storage to /var/lib/etcd using CIFS
Before=etcd3.service

[Mount]
What=//8tnh2etcd.file.core.windows.net/etcd
Where=/var/lib/etcd
Type=cifs
Options=nofail,vers=3.0,credentials=/etc/smbcredentials/8tnh2etcd.cred,serverino

[Install]
WantedBy=multi-user.target
`
