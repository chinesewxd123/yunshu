package service

import (
	"context"
	"strings"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
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
				host := ""
				if len(ins.PublicIpAddress.IpAddress) > 0 {
					host = strings.TrimSpace(ins.PublicIpAddress.IpAddress[0])
				}
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
				out = append(out, CloudInstance{
					InstanceID: ins.InstanceId,
					Name:       strings.TrimSpace(ins.InstanceName),
					Host:       host,
					Region:     region,
					OSType:     osType,
					Status:     1,
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
