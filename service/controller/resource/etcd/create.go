package etcd

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v3/service/controller/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	dnsClient, err := r.getDNSRecordSetsClient(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	ips, err := r.getVMSSPrivateIPs(ctx, key.ResourceGroupName(cr), key.MasterVMSSName(cr))
	if err != nil {
		return microerror.Mask(err)
	}

	zone := fmt.Sprintf("%s.k8s.%s", key.ClusterID(cr), key.DNSZoneAPI(cr))

	var serverSrvRecords []dns.SrvRecord
	var clientSrvRecords []dns.SrvRecord

	for machineName, ip := range ips {
		var params dns.RecordSet
		{
			a := dns.ARecord{Ipv4Address: &ip}

			params.RecordSetProperties = &dns.RecordSetProperties{
				TTL:      to.Int64Ptr(60),
				ARecords: &[]dns.ARecord{a},
			}
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Ensuring dns record A %s => %s", machineName, ip))

		_, err = dnsClient.CreateOrUpdate(ctx, key.ResourceGroupName(cr), zone, machineName, dns.A, params, "", "")
		if err != nil {
			return microerror.Mask(err)
		}

		serverSrvRecords = append(serverSrvRecords, dns.SrvRecord{
			Priority: to.Int32Ptr(0),
			Weight:   to.Int32Ptr(0),
			Port:     to.Int32Ptr(2380),
			Target:   to.StringPtr(fmt.Sprintf("%s.%s", machineName, zone)),
		})

		clientSrvRecords = append(clientSrvRecords, dns.SrvRecord{
			Priority: to.Int32Ptr(0),
			Weight:   to.Int32Ptr(0),
			Port:     to.Int32Ptr(2379),
			Target:   to.StringPtr(fmt.Sprintf("%s.%s", machineName, zone)),
		})
	}

	var serverSrvProperties dns.RecordSet
	{
		serverSrvProperties.RecordSetProperties = &dns.RecordSetProperties{
			TTL:        to.Int64Ptr(60),
			SrvRecords: &serverSrvRecords,
		}
	}

	name := "_etcd-server-ssl._tcp"

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Ensuring dns record SRV %s.%s", name, zone))

	_, err = dnsClient.CreateOrUpdate(ctx, key.ResourceGroupName(cr), zone, name, dns.SRV, serverSrvProperties, "", "")
	if err != nil {
		return microerror.Mask(err)
	}

	name = "_etcd-client-ssl._tcp"

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Ensuring dns record SRV %s.%s", name, zone))

	var clientSrvProperties dns.RecordSet
	{
		clientSrvProperties.RecordSetProperties = &dns.RecordSetProperties{
			TTL:        to.Int64Ptr(60),
			SrvRecords: &clientSrvRecords,
		}
	}

	_, err = dnsClient.CreateOrUpdate(ctx, key.ResourceGroupName(cr), zone, name, dns.SRV, clientSrvProperties, "", "")
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
