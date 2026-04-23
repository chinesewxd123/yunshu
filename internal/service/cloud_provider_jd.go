package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	jdcore "github.com/jdcloud-api/jdcloud-sdk-go/core"
	jddiskmodels "github.com/jdcloud-api/jdcloud-sdk-go/services/disk/models"
	jdtagapi "github.com/jdcloud-api/jdcloud-sdk-go/services/resourcetag/apis"
	jdtagclient "github.com/jdcloud-api/jdcloud-sdk-go/services/resourcetag/client"
	jdtagmodels "github.com/jdcloud-api/jdcloud-sdk-go/services/resourcetag/models"
	jdvmapi "github.com/jdcloud-api/jdcloud-sdk-go/services/vm/apis"
	jdvmclient "github.com/jdcloud-api/jdcloud-sdk-go/services/vm/client"
	jdvmmodels "github.com/jdcloud-api/jdcloud-sdk-go/services/vm/models"
)

type JdCloudProvider struct{}

func (p *JdCloudProvider) ListInstances(ctx context.Context, ak, sk, regionScope string) ([]CloudInstance, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	regions := make([]string, 0)
	for _, it := range strings.Split(regionScope, ",") {
		v := strings.TrimSpace(it)
		if v != "" {
			regions = append(regions, v)
		}
	}
	if len(regions) == 0 {
		// 京东云默认地域：华北-北京
		regions = []string{"cn-north-1"}
	}

	out := make([]CloudInstance, 0)
	for _, region := range regions {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		credential := jdcore.NewCredentials(strings.TrimSpace(ak), strings.TrimSpace(sk))
		client := jdvmclient.NewVmClient(credential)
		pageNumber := 1
		pageSize := 100
		for {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			req := jdvmapi.NewDescribeInstancesRequest(region)
			req.SetPageNumber(pageNumber)
			req.SetPageSize(pageSize)
			resp, err := client.DescribeInstances(req)
			if err != nil {
				return nil, err
			}
			for _, ins := range resp.Result.Instances {
				publicIP := strings.TrimSpace(ins.ElasticIpAddress)
				privateIP := strings.TrimSpace(ins.PrivateIpAddress)
				host := publicIP
				if host == "" {
					host = privateIP
				}
				if host == "" {
					continue
				}
				osType := strings.ToLower(strings.TrimSpace(ins.OsType))
				if osType == "" {
					osType = "linux"
				}
				statusText := strings.TrimSpace(ins.Status)
				status := 0
				if strings.EqualFold(statusText, "running") {
					status = 1
				}
				networkInfo := ""
				if strings.TrimSpace(ins.VpcId) != "" || strings.TrimSpace(ins.SubnetId) != "" {
					networkInfo = fmt.Sprintf("vpc:%s subnet:%s", strings.TrimSpace(ins.VpcId), strings.TrimSpace(ins.SubnetId))
				}
				configInfo := fmt.Sprintf("%s %s", strings.TrimSpace(ins.InstanceType), formatJdDiskInfo(ins.SystemDisk, ins.DataDisks))
				tagsJSON := marshalJdTags(ins.Tags)
				out = append(out, CloudInstance{
					InstanceID:        strings.TrimSpace(ins.InstanceId),
					Name:              strings.TrimSpace(ins.InstanceName),
					Host:              host,
					Region:            region,
					Zone:              strings.TrimSpace(ins.Az),
					Spec:              strings.TrimSpace(ins.InstanceType),
					ConfigInfo:        configInfo,
					OSName:            strings.TrimSpace(ins.OsType),
					NetworkInfo:       networkInfo,
					ChargeType:        strings.TrimSpace(ins.Charge.ChargeMode),
					NetworkChargeType: strings.TrimSpace(ins.Charge.ChargeStatus),
					TagsJSON:          tagsJSON,
					PublicIP:          publicIP,
					PrivateIP:         privateIP,
					StatusText:        statusText,
					OSType:            osType,
					Status:            status,
				})
			}
			total := resp.Result.TotalCount
			if pageNumber*pageSize >= total {
				break
			}
			pageNumber++
		}
	}
	return out, nil
}

func marshalJdTags(tags []jddiskmodels.Tag) string {
	if len(tags) == 0 {
		return ""
	}
	obj := make(map[string]string, len(tags))
	for _, t := range tags {
		if t.Key == nil {
			continue
		}
		k := strings.TrimSpace(*t.Key)
		if k == "" {
			continue
		}
		v := ""
		if t.Value != nil {
			v = strings.TrimSpace(*t.Value)
		}
		obj[k] = v
	}
	if len(obj) == 0 {
		return ""
	}
	b, err := json.Marshal(obj)
	if err != nil {
		return ""
	}
	return string(b)
}

func (p *JdCloudProvider) ResetInstancePassword(ctx context.Context, ak, sk, region, instanceID, newPassword string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	credential := jdcore.NewCredentials(strings.TrimSpace(ak), strings.TrimSpace(sk))
	client := jdvmclient.NewVmClient(credential)
	req := jdvmapi.NewModifyInstancePasswordRequest(strings.TrimSpace(region), strings.TrimSpace(instanceID), newPassword)
	_, err := client.ModifyInstancePassword(req)
	return err
}

func (p *JdCloudProvider) RebootInstance(ctx context.Context, ak, sk, region, instanceID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	credential := jdcore.NewCredentials(strings.TrimSpace(ak), strings.TrimSpace(sk))
	client := jdvmclient.NewVmClient(credential)
	req := jdvmapi.NewRebootInstanceRequest(strings.TrimSpace(region), strings.TrimSpace(instanceID))
	_, err := client.RebootInstance(req)
	return err
}

func (p *JdCloudProvider) ShutdownInstance(ctx context.Context, ak, sk, region, instanceID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	credential := jdcore.NewCredentials(strings.TrimSpace(ak), strings.TrimSpace(sk))
	client := jdvmclient.NewVmClient(credential)
	req := jdvmapi.NewStopInstanceRequest(strings.TrimSpace(region), strings.TrimSpace(instanceID))
	_, err := client.StopInstance(req)
	return err
}

func (p *JdCloudProvider) QueryInstanceExpireAt(ctx context.Context, ak, sk, region, instanceID string) (*time.Time, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	credential := jdcore.NewCredentials(strings.TrimSpace(ak), strings.TrimSpace(sk))
	client := jdvmclient.NewVmClient(credential)
	req := jdvmapi.NewDescribeInstanceRequest(strings.TrimSpace(region), strings.TrimSpace(instanceID))
	resp, err := client.DescribeInstance(req)
	if err != nil {
		return nil, err
	}
	raw := strings.TrimSpace(resp.Result.Instance.Charge.ChargeExpiredTime)
	if raw == "" {
		return nil, nil
	}
	if t, parseErr := time.Parse(time.RFC3339, raw); parseErr == nil {
		return &t, nil
	}
	return nil, nil
}

func (p *JdCloudProvider) SyncInstanceTags(ctx context.Context, ak, sk, region, instanceID string, oldTags, newTags map[string]string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	credential := jdcore.NewCredentials(strings.TrimSpace(ak), strings.TrimSpace(sk))
	client := jdtagclient.NewResourcetagClient(credential)
	resources := []jdtagmodels.ResourcesMap{
		{
			ServiceCode: "vm",
			ResourceId:  []string{strings.TrimSpace(instanceID)},
		},
	}
	if len(oldTags) == 0 && len(newTags) == 0 {
		return nil
	}
	toUnbind := make([]jdtagmodels.Tag, 0)
	for k, oldV := range oldTags {
		nv, ok := newTags[k]
		if !ok || strings.TrimSpace(nv) != strings.TrimSpace(oldV) {
			key := strings.TrimSpace(k)
			if key != "" {
				toUnbind = append(toUnbind, jdtagmodels.Tag{Key: key, Value: strings.TrimSpace(oldV)})
			}
		}
	}
	if len(toUnbind) > 0 {
		req := jdtagapi.NewUnTagResourcesRequest(strings.TrimSpace(region), &jdtagmodels.UnTagResourcesReqVo{
			Resources: resources,
			Tags:      toUnbind,
		})
		if _, err := client.UnTagResources(req); err != nil {
			return err
		}
	}
	toBind := make([]jdtagmodels.Tag, 0)
	for k, v := range newTags {
		key := strings.TrimSpace(k)
		if key == "" {
			continue
		}
		nv := strings.TrimSpace(v)
		if ov, ok := oldTags[key]; ok && strings.TrimSpace(ov) == nv {
			continue
		}
		toBind = append(toBind, jdtagmodels.Tag{Key: key, Value: nv})
	}
	if len(toBind) > 0 {
		req := jdtagapi.NewTagResourcesRequest(strings.TrimSpace(region), &jdtagmodels.TagResourcesReqVo{
			Resources: resources,
			Tags:      toBind,
		})
		if _, err := client.TagResources(req); err != nil {
			return err
		}
	}
	return nil
}

func formatJdDiskInfo(systemDisk jdvmmodels.InstanceDiskAttachment, dataDisks []jdvmmodels.InstanceDiskAttachment) string {
	systemSize := 0
	if systemDisk.CloudDisk.DiskSizeGB > 0 {
		systemSize = systemDisk.CloudDisk.DiskSizeGB
	}
	dataCount := 0
	dataTotal := 0
	for _, d := range dataDisks {
		if d.CloudDisk.DiskSizeGB <= 0 {
			continue
		}
		dataCount++
		dataTotal += d.CloudDisk.DiskSizeGB
	}
	return fmt.Sprintf("系统盘:%dGiB 数据盘:%d块/%dGiB", systemSize, dataCount, dataTotal)
}
