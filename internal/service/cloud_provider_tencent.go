package service

import (
	"yunshu/internal/service/svcerr"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	sts "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sts/v20180813"
	tag "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tag/v20180813"
)

type TencentCloudProvider struct{}

func (p *TencentCloudProvider) ListInstances(ctx context.Context, ak, sk, regionScope string) ([]CloudInstance, error) {
	if err := ctx.Err(); err != nil {
		return nil, svcerr.Pass(ctx, "project.cloud", "ListInstances", err)
	}
	regions := make([]string, 0)
	for _, it := range strings.Split(regionScope, ",") {
		v := strings.TrimSpace(it)
		if v == "" {
			continue
		}
		if api := tencentAPIRegionFromUserInput(v); api != "" {
			regions = append(regions, api)
			continue
		}
		lv := strings.ToLower(v)
		if strings.HasPrefix(lv, "ap-") || strings.HasPrefix(lv, "na-") || strings.HasPrefix(lv, "eu-") {
			regions = append(regions, lv)
			continue
		}
		// 无法识别的 token 仍尝试原样（兼容自定义地域写法）
		regions = append(regions, v)
	}
	if len(regions) == 0 {
		// 腾讯云默认地域：广州
		regions = []string{"ap-guangzhou"}
	}

	out := make([]CloudInstance, 0)
	for _, region := range regions {
		if err := ctx.Err(); err != nil {
			return nil, svcerr.Pass(ctx, "project.cloud", "ListInstances", err)
		}
		cred := common.NewCredential(strings.TrimSpace(ak), strings.TrimSpace(sk))
		cpf := profile.NewClientProfile()
		cpf.HttpProfile.Scheme = "https"
		client, err := cvm.NewClient(cred, region, cpf)
		if err != nil {
			return nil, svcerr.Pass(ctx, "project.cloud", "ListInstances", err)
		}
		offset := int64(0)
		limit := int64(100)
		for {
			if err := ctx.Err(); err != nil {
				return nil, svcerr.Pass(ctx, "project.cloud", "ListInstances", err)
			}
			req := cvm.NewDescribeInstancesRequest()
			req.Limit = common.Int64Ptr(limit)
			req.Offset = common.Int64Ptr(offset)
			resp, err := client.DescribeInstances(req)
			if err != nil {
				return nil, svcerr.Pass(ctx, "project.cloud", "ListInstances", err)
			}
			if resp == nil || resp.Response == nil {
				break
			}
			for _, ins := range resp.Response.InstanceSet {
				if ins == nil {
					continue
				}
				instanceID := ""
				if ins.InstanceId != nil {
					instanceID = strings.TrimSpace(*ins.InstanceId)
				}
				host := ""
				publicIP := ""
				privateIP := ""
				if len(ins.PublicIpAddresses) > 0 && ins.PublicIpAddresses[0] != nil {
					publicIP = strings.TrimSpace(*ins.PublicIpAddresses[0])
					host = publicIP
				}
				if host == "" && len(ins.PrivateIpAddresses) > 0 && ins.PrivateIpAddresses[0] != nil {
					privateIP = strings.TrimSpace(*ins.PrivateIpAddresses[0])
					host = privateIP
				}
				if privateIP == "" && len(ins.PrivateIpAddresses) > 0 && ins.PrivateIpAddresses[0] != nil {
					privateIP = strings.TrimSpace(*ins.PrivateIpAddresses[0])
				}
				if host == "" {
					// 无公网/私网 IP 的包年实例仍可能有过期时间，需参与到期评估
					if instanceID != "" {
						host = instanceID
					} else {
						continue
					}
				}
				osName := ""
				if ins.OsName != nil {
					osName = strings.TrimSpace(*ins.OsName)
				}
				osType := "linux"
				if strings.Contains(strings.ToLower(osName), "windows") {
					osType = "windows"
				}
				status := int64(1)
				if ins.InstanceState != nil {
					// RUNNING / STOPPED 等
					if strings.ToUpper(strings.TrimSpace(*ins.InstanceState)) != "RUNNING" {
						status = 0
					}
				}
				name := ""
				if ins.InstanceName != nil {
					name = strings.TrimSpace(*ins.InstanceName)
				}
				zone := ""
				if ins.Placement != nil && ins.Placement.Zone != nil {
					zone = strings.TrimSpace(*ins.Placement.Zone)
				}
				spec := ""
				if ins.InstanceType != nil {
					spec = strings.TrimSpace(*ins.InstanceType)
				}
				statusText := ""
				if ins.InstanceState != nil {
					statusText = strings.TrimSpace(*ins.InstanceState)
				}
				osName = strings.TrimSpace(osName)
				configInfo := ""
				if ins.CPU != nil || ins.Memory != nil {
					configInfo = fmt.Sprintf("CPU:%v核 Memory:%vGB", valueOrZeroInt64(ins.CPU), valueOrZeroInt64(ins.Memory))
				}
				configInfo = strings.TrimSpace(configInfo + " " + formatTencentDiskInfo(ins.SystemDisk, ins.DataDisks))
				networkInfo := ""
				if ins.VirtualPrivateCloud != nil {
					vpcID := strings.TrimSpace(valueOrEmpty(ins.VirtualPrivateCloud.VpcId))
					subnetID := strings.TrimSpace(valueOrEmpty(ins.VirtualPrivateCloud.SubnetId))
					if vpcID != "" || subnetID != "" {
						networkInfo = fmt.Sprintf("vpc:%s subnet:%s", vpcID, subnetID)
					}
				}
				chargeType := strings.TrimSpace(valueOrEmpty(ins.InstanceChargeType))
				networkChargeType := ""
				if ins.InternetAccessible != nil {
					networkChargeType = strings.TrimSpace(valueOrEmpty(ins.InternetAccessible.InternetChargeType))
				}
				tagsJSON := marshalTencentTags(ins.Tags)
				out = append(out, CloudInstance{
					InstanceID:        instanceID,
					Name:              name,
					Host:              host,
					Region:            region,
					Zone:              zone,
					Spec:              spec,
					ConfigInfo:        configInfo,
					OSName:            osName,
					NetworkInfo:       networkInfo,
					ChargeType:        chargeType,
					NetworkChargeType: networkChargeType,
					TagsJSON:          tagsJSON,
					PublicIP:          publicIP,
					PrivateIP:         privateIP,
					StatusText:        statusText,
					OSType:            osType,
					Status:            int(status),
				})
			}
			total := int64(0)
			if resp.Response.TotalCount != nil {
				total = *resp.Response.TotalCount
			}
			offset += limit
			if offset >= total {
				break
			}
		}
	}
	return out, nil
}

func valueOrEmpty(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func valueOrZeroInt64(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}

func marshalTencentTags(tags []*cvm.Tag) string {
	if len(tags) == 0 {
		return ""
	}
	obj := make(map[string]string, len(tags))
	for _, t := range tags {
		if t == nil {
			continue
		}
		k := strings.TrimSpace(valueOrEmpty(t.Key))
		if k == "" {
			continue
		}
		obj[k] = strings.TrimSpace(valueOrEmpty(t.Value))
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

func (p *TencentCloudProvider) ResetInstancePassword(ctx context.Context, ak, sk, region, instanceID, newPassword string) error {
	if err := ctx.Err(); err != nil {
		return svcerr.Pass(ctx, "project.cloud", "ResetInstancePassword", err)
	}
	cred := common.NewCredential(strings.TrimSpace(ak), strings.TrimSpace(sk))
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Scheme = "https"
	client, err := cvm.NewClient(cred, strings.TrimSpace(region), cpf)
	if err != nil {
		return svcerr.Pass(ctx, "project.cloud", "ResetInstancePassword", err)
	}
	req := cvm.NewResetInstancesPasswordRequest()
	req.InstanceIds = []*string{common.StringPtr(strings.TrimSpace(instanceID))}
	req.Password = common.StringPtr(newPassword)
	req.ForceStop = common.BoolPtr(true)
	_, err = client.ResetInstancesPassword(req)
	return svcerr.Pass(ctx, "project.cloud", "ResetInstancePassword", err)
}

func (p *TencentCloudProvider) RebootInstance(ctx context.Context, ak, sk, region, instanceID string) error {
	if err := ctx.Err(); err != nil {
		return svcerr.Pass(ctx, "project.cloud", "RebootInstance", err)
	}
	cred := common.NewCredential(strings.TrimSpace(ak), strings.TrimSpace(sk))
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Scheme = "https"
	client, err := cvm.NewClient(cred, strings.TrimSpace(region), cpf)
	if err != nil {
		return svcerr.Pass(ctx, "project.cloud", "RebootInstance", err)
	}
	req := cvm.NewRebootInstancesRequest()
	req.InstanceIds = []*string{common.StringPtr(strings.TrimSpace(instanceID))}
	req.StopType = common.StringPtr("SOFT_FIRST")
	_, err = client.RebootInstances(req)
	return svcerr.Pass(ctx, "project.cloud", "RebootInstance", err)
}

func (p *TencentCloudProvider) ShutdownInstance(ctx context.Context, ak, sk, region, instanceID string) error {
	if err := ctx.Err(); err != nil {
		return svcerr.Pass(ctx, "project.cloud", "ShutdownInstance", err)
	}
	cred := common.NewCredential(strings.TrimSpace(ak), strings.TrimSpace(sk))
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Scheme = "https"
	client, err := cvm.NewClient(cred, strings.TrimSpace(region), cpf)
	if err != nil {
		return svcerr.Pass(ctx, "project.cloud", "ShutdownInstance", err)
	}
	req := cvm.NewStopInstancesRequest()
	req.InstanceIds = []*string{common.StringPtr(strings.TrimSpace(instanceID))}
	req.StopType = common.StringPtr("SOFT_FIRST")
	_, err = client.StopInstances(req)
	return svcerr.Pass(ctx, "project.cloud", "ShutdownInstance", err)
}

func (p *TencentCloudProvider) QueryInstanceExpireAt(ctx context.Context, ak, sk, region, instanceID string) (*time.Time, error) {
	if err := ctx.Err(); err != nil {
		return nil, svcerr.Pass(ctx, "project.cloud", "QueryInstanceExpireAt", err)
	}
	cred := common.NewCredential(strings.TrimSpace(ak), strings.TrimSpace(sk))
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Scheme = "https"
	client, err := cvm.NewClient(cred, strings.TrimSpace(region), cpf)
	if err != nil {
		return nil, svcerr.Pass(ctx, "project.cloud", "QueryInstanceExpireAt", err)
	}
	req := cvm.NewDescribeInstancesRequest()
	req.Limit = common.Int64Ptr(1)
	req.Filters = []*cvm.Filter{
		{
			Name:   common.StringPtr("instance-id"),
			Values: []*string{common.StringPtr(strings.TrimSpace(instanceID))},
		},
	}
	resp, err := client.DescribeInstances(req)
	if err != nil {
		return nil, svcerr.Pass(ctx, "project.cloud", "QueryInstanceExpireAt", err)
	}
	if resp == nil || resp.Response == nil || len(resp.Response.InstanceSet) == 0 {
		return nil, nil
	}
	ins := resp.Response.InstanceSet[0]
	if ins == nil || ins.ExpiredTime == nil {
		return nil, nil
	}
	raw := strings.TrimSpace(*ins.ExpiredTime)
	if raw == "" {
		return nil, nil
	}
	if t, ok := parseTencentExpiredTime(raw); ok {
		return &t, nil
	}
	return nil, nil
}

func parseTencentExpiredTime(raw string) (time.Time, bool) {
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t, true
	}
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return t, true
	}
	layouts := []string{
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04Z",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, raw, time.Local); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func (p *TencentCloudProvider) SyncInstanceTags(ctx context.Context, ak, sk, region, instanceID string, oldTags, newTags map[string]string) error {
	if err := ctx.Err(); err != nil {
		return svcerr.Pass(ctx, "project.cloud", "SyncInstanceTags", err)
	}
	cred := common.NewCredential(strings.TrimSpace(ak), strings.TrimSpace(sk))
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Scheme = "https"
	resource, err := p.buildTagResource(ctx, cred, cpf, strings.TrimSpace(region), strings.TrimSpace(instanceID))
	if err != nil {
		return svcerr.Pass(ctx, "project.cloud", "SyncInstanceTags", err)
	}
	if len(oldTags) == 0 && len(newTags) == 0 {
		return nil
	}
	tagClient, err := tag.NewClient(cred, strings.TrimSpace(region), cpf)
	if err != nil {
		return svcerr.Pass(ctx, "project.cloud", "SyncInstanceTags", err)
	}
	toUnbind := make([]*string, 0)
	for k, oldV := range oldTags {
		nv, ok := newTags[k]
		if !ok || strings.TrimSpace(nv) != strings.TrimSpace(oldV) {
			key := strings.TrimSpace(k)
			if key != "" {
				toUnbind = append(toUnbind, common.StringPtr(key))
			}
		}
	}
	if len(toUnbind) > 0 {
		req := tag.NewUnTagResourcesRequest()
		req.ResourceList = []*string{common.StringPtr(resource)}
		req.TagKeys = toUnbind
		if _, err := tagClient.UnTagResources(req); err != nil {
			return svcerr.Pass(ctx, "project.cloud", "SyncInstanceTags", err)
		}
	}
	toBind := make([]*tag.Tag, 0)
	for k, v := range newTags {
		key := strings.TrimSpace(k)
		if key == "" {
			continue
		}
		nv := strings.TrimSpace(v)
		if ov, ok := oldTags[key]; ok && strings.TrimSpace(ov) == nv {
			continue
		}
		toBind = append(toBind, &tag.Tag{
			TagKey:   common.StringPtr(key),
			TagValue: common.StringPtr(nv),
		})
	}
	if len(toBind) > 0 {
		req := tag.NewTagResourcesRequest()
		req.ResourceList = []*string{common.StringPtr(resource)}
		req.Tags = toBind
		if _, err := tagClient.TagResources(req); err != nil {
			return svcerr.Pass(ctx, "project.cloud", "SyncInstanceTags", err)
		}
	}
	return nil
}

func (p *TencentCloudProvider) buildTagResource(ctx context.Context, cred *common.Credential, cpf *profile.ClientProfile, region, instanceID string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", svcerr.Pass(ctx, "project.cloud", "buildTagResource", err)
	}
	stsClient, err := sts.NewClient(cred, region, cpf)
	if err != nil {
		return "", svcerr.Pass(ctx, "project.cloud", "buildTagResource", err)
	}
	identityResp, err := stsClient.GetCallerIdentity(sts.NewGetCallerIdentityRequest())
	if err != nil {
		return "", svcerr.Pass(ctx, "project.cloud", "buildTagResource", err)
	}
	if identityResp == nil || identityResp.Response == nil || identityResp.Response.AccountId == nil {
		return "", fmt.Errorf("无法获取腾讯云账号 AccountId")
	}
	accountID := strings.TrimSpace(*identityResp.Response.AccountId)
	if accountID == "" {
		return "", fmt.Errorf("腾讯云账号 AccountId 为空")
	}
	return fmt.Sprintf("qcs::cvm:%s:uin/%s:instance/%s", region, accountID, instanceID), nil
}

func formatTencentDiskInfo(systemDisk *cvm.SystemDisk, dataDisks []*cvm.DataDisk) string {
	system := int64(0)
	if systemDisk != nil && systemDisk.DiskSize != nil {
		system = *systemDisk.DiskSize
	}
	dataCount := 0
	dataTotal := int64(0)
	for _, d := range dataDisks {
		if d == nil || d.DiskSize == nil {
			continue
		}
		dataCount++
		dataTotal += *d.DiskSize
	}
	return fmt.Sprintf("系统盘:%dGiB 数据盘:%d块/%dGiB", system, dataCount, dataTotal)
}
