package ignition

const EtcdMountUnit = `[Unit]
Description=Mounts azure file storage to /var/lib/etcd using CIFS
Before=etcd3.service

[Mount]
What=//{{.Cluster.ID}}etcd.file.core.windows.net/etcd
Where=/var/lib/etcd
Type=cifs
Options=nofail,vers=3.0,credentials=/etc/smbcredentials/{{.Cluster.ID}}etcd.cred,serverino

[Install]
WantedBy=multi-user.target
`
