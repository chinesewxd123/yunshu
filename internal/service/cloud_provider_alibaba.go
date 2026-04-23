package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
)

type AlibabaCloudProvider struct{}

func (p *AlibabaCloudProvider) ListInstances(ctx context.Context, ak, sk, regionScope string) ([]CloudInstance, error) {
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
		regions = []string{"cn-hangzhou"}
	}

	out := make([]CloudInstance, 0)
	for _, region := range regions {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		client, err := ecs.NewClientWithAccessKey(region, ak, sk)
		if err != nil {
			return nil, err
		}
		pageNumber := 1
		for {
			req := ecs.CreateDescribeInstancesRequest()
			req.Scheme = "https"
			req.PageSize = requests.NewInteger(100)
			req.PageNumber = requests.NewInteger(pageNumber)
			resp, err := client.DescribeInstances(req)
			if err != nil {
				return nil, err
			}
			for _, ins := range resp.Instances.Instance {
				publicIP := ""
				privateIP := ""
				if len(ins.PublicIpAddress.IpAddress) > 0 {
					publicIP = strings.TrimSpace(ins.PublicIpAddress.IpAddress[0])
				}
				if len(ins.InnerIpAddress.IpAddress) > 0 {
					privateIP = strings.TrimSpace(ins.InnerIpAddress.IpAddress[0])
				}
				host := publicIP
				if host == "" {
					if len(ins.InnerIpAddress.IpAddress) > 0 {
						host = strings.TrimSpace(ins.InnerIpAddress.IpAddress[0])
					}
				}
				if host == "" {
					continue
				}
				osType := "linux"
				if strings.Contains(strings.ToLower(ins.OSNameEn), "windows") {
					osType = "windows"
				}
				diskInfo := p.describeInstanceDiskSummary(ctx, client, strings.TrimSpace(ins.InstanceId))
				networkInfo := ""
				if vpc := strings.TrimSpace(ins.VpcAttributes.VpcId); vpc != "" || strings.TrimSpace(ins.VpcAttributes.VSwitchId) != "" {
					networkInfo = fmt.Sprintf("vpc:%s vsw:%s", strings.TrimSpace(ins.VpcAttributes.VpcId), strings.TrimSpace(ins.VpcAttributes.VSwitchId))
				}
				tagsJSON := marshalAlibabaTags(ins.Tags.Tag)
				out = append(out, CloudInstance{
					InstanceID:        ins.InstanceId,
					Name:              strings.TrimSpace(ins.InstanceName),
					Host:              host,
					Region:            region,
					Zone:              strings.TrimSpace(ins.ZoneId),
					Spec:              strings.TrimSpace(ins.InstanceType),
					ConfigInfo:        fmt.Sprintf("CPU:%d核 Memory:%dMB %s", ins.CPU, ins.Memory, diskInfo),
					OSName:            strings.TrimSpace(ins.OSName),
					NetworkInfo:       networkInfo,
					ChargeType:        strings.TrimSpace(ins.InstanceChargeType),
					NetworkChargeType: strings.TrimSpace(ins.InternetChargeType),
					TagsJSON:          tagsJSON,
					PublicIP:          publicIP,
					PrivateIP:         privateIP,
					StatusText:        strings.TrimSpace(ins.Status),
					OSType:            osType,
					Status: func() int {
						if strings.EqualFold(strings.TrimSpace(ins.Status), "Running") {
							return 1
						}
						return 0
					}(),
				})
			}
			total := resp.TotalCount
			if pageNumber*100 >= total {
				break
			}
			pageNumber++
		}
	}
	return out, nil
}

func marshalAlibabaTags(tags []ecs.Tag) string {
	if len(tags) == 0 {
		return ""
	}
	obj := make(map[string]string, len(tags))
	for _, t := range tags {
		k := strings.TrimSpace(t.Key)
		if k == "" {
			k = strings.TrimSpace(t.TagKey)
		}
		if k == "" {
			continue
		}
		v := strings.TrimSpace(t.Value)
		if v == "" {
			v = strings.TrimSpace(t.TagValue)
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

func (p *AlibabaCloudProvider) ResetInstancePassword(ctx context.Context, ak, sk, region, instanceID, newPassword string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	client, err := ecs.NewClientWithAccessKey(strings.TrimSpace(region), strings.TrimSpace(ak), strings.TrimSpace(sk))
	if err != nil {
		return err
	}
	req := ecs.CreateModifyInstanceAttributeRequest()
	req.Scheme = "https"
	req.InstanceId = strings.TrimSpace(instanceID)
	req.Password = newPassword
	_, err = client.ModifyInstanceAttribute(req)
	return err
}

func (p *AlibabaCloudProvider) RebootInstance(ctx context.Context, ak, sk, region, instanceID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	client, err := ecs.NewClientWithAccessKey(strings.TrimSpace(region), strings.TrimSpace(ak), strings.TrimSpace(sk))
	if err != nil {
		return err
	}
	req := ecs.CreateRebootInstanceRequest()
	req.Scheme = "https"
	req.InstanceId = strings.TrimSpace(instanceID)
	req.ForceStop = requests.NewBoolean(true)
	_, err = client.RebootInstance(req)
	return err
}

func (p *AlibabaCloudProvider) ShutdownInstance(ctx context.Context, ak, sk, region, instanceID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	client, err := ecs.NewClientWithAccessKey(strings.TrimSpace(region), strings.TrimSpace(ak), strings.TrimSpace(sk))
	if err != nil {
		return err
	}
	req := ecs.CreateStopInstanceRequest()
	req.Scheme = "https"
	req.InstanceId = strings.TrimSpace(instanceID)
	req.ForceStop = requests.NewBoolean(true)
	_, err = client.StopInstance(req)
	return err
}

func (p *AlibabaCloudProvider) QueryInstanceExpireAt(ctx context.Context, ak, sk, region, instanceID string) (*time.Time, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	client, err := ecs.NewClientWithAccessKey(strings.TrimSpace(region), strings.TrimSpace(ak), strings.TrimSpace(sk))
	if err != nil {
		return nil, err
	}
	req := ecs.CreateDescribeInstancesRequest()
	req.Scheme = "https"
	req.PageSize = requests.NewInteger(1)
	req.InstanceIds = `["` + strings.TrimSpace(instanceID) + `"]`
	resp, err := client.DescribeInstances(req)
	if err != nil {
		return nil, err
	}
	if len(resp.Instances.Instance) == 0 {
		return nil, nil
	}
	raw := strings.TrimSpace(resp.Instances.Instance[0].ExpiredTime)
	if raw == "" {
		return nil, nil
	}
	if t, parseErr := time.Parse(time.RFC3339, raw); parseErr == nil {
		return &t, nil
	}
	if t, parseErr := time.Parse("2006-01-02T15:04Z", raw); parseErr == nil {
		return &t, nil
	}
	return nil, nil
}

func (p *AlibabaCloudProvider) SyncInstanceTags(ctx context.Context, ak, sk, region, instanceID string, oldTags, newTags map[string]string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	client, err := ecs.NewClientWithAccessKey(strings.TrimSpace(region), strings.TrimSpace(ak), strings.TrimSpace(sk))
	if err != nil {
		return err
	}
	if len(oldTags) == 0 && len(newTags) == 0 {
		return nil
	}
	resourceType := "instance"
	resourceIDs := []string{strings.TrimSpace(instanceID)}
	toUnbind := make([]string, 0)
	for k, oldV := range oldTags {
		nv, ok := newTags[k]
		if !ok || strings.TrimSpace(nv) != strings.TrimSpace(oldV) {
			key := strings.TrimSpace(k)
			if key != "" {
				toUnbind = append(toUnbind, key)
			}
		}
	}
	if len(toUnbind) > 0 {
		req := ecs.CreateUntagResourcesRequest()
		req.Scheme = "https"
		req.ResourceType = resourceType
		req.ResourceId = &resourceIDs
		req.TagKey = &toUnbind
		if _, err := client.UntagResources(req); err != nil {
			return err
		}
	}
	toBind := make([]ecs.TagResourcesTag, 0)
	for k, v := range newTags {
		key := strings.TrimSpace(k)
		if key == "" {
			continue
		}
		nv := strings.TrimSpace(v)
		if ov, ok := oldTags[key]; ok && strings.TrimSpace(ov) == nv {
			continue
		}
		toBind = append(toBind, ecs.TagResourcesTag{Key: key, Value: nv})
	}
	if len(toBind) > 0 {
		req := ecs.CreateTagResourcesRequest()
		req.Scheme = "https"
		req.ResourceType = resourceType
		req.ResourceId = &resourceIDs
		req.Tag = &toBind
		if _, err := client.TagResources(req); err != nil {
			return err
		}
	}
	return nil
}

func (p *AlibabaCloudProvider) describeInstanceDiskSummary(ctx context.Context, client *ecs.Client, instanceID string) string {
	if strings.TrimSpace(instanceID) == "" || client == nil {
		return "系统盘:- 数据盘:-"
	}
	req := ecs.CreateDescribeDisksRequest()
	req.Scheme = "https"
	req.InstanceId = instanceID
	req.PageSize = requests.NewInteger(100)
	req.PageNumber = requests.NewInteger(1)
	resp, err := client.DescribeDisks(req)
	if err != nil || resp == nil {
		return "系统盘:- 数据盘:-"
	}
	systemSize := 0
	dataTotal := 0
	dataCount := 0
	for _, d := range resp.Disks.Disk {
		dType := strings.ToLower(strings.TrimSpace(d.Type))
		if dType == "system" {
			systemSize = d.Size
			continue
		}
		if dType == "data" {
			dataCount++
			dataTotal += d.Size
		}
	}
	return fmt.Sprintf("系统盘:%dGiB 数据盘:%d块/%dGiB", systemSize, dataCount, dataTotal)
}
