package agent

import (
	"context"
	"strings"

	pb "yunshu/internal/grpc/proto"
)

const discoveryRootLogType = "_discovery_root"

type runtimeConfigBundle struct {
	Sources   []runtimeSource
	ProjectID uint
	Roots     []string
}

func fetchRuntimeConfig(ctx context.Context, cli pb.AgentRuntimeServiceClient, token string) (runtimeConfigBundle, error) {
	out, err := cli.GetRuntimeConfig(ctx, &pb.GetRuntimeConfigRequest{Token: token})
	if err != nil {
		return runtimeConfigBundle{}, err
	}
	sources := make([]runtimeSource, 0, len(out.GetSources()))
	roots := make([]string, 0, 4)
	for _, it := range out.GetSources() {
		if strings.EqualFold(strings.TrimSpace(it.GetLogType()), discoveryRootLogType) {
			if p := strings.TrimSpace(it.GetPath()); p != "" {
				roots = append(roots, p)
			}
			continue
		}
		sources = append(sources, runtimeSource{
			LogSourceID: uint(it.GetLogSourceId()),
			LogType:     it.GetLogType(),
			Path:        it.GetPath(),
		})
	}
	return runtimeConfigBundle{
		Sources:   sources,
		ProjectID: uint(out.GetProjectId()),
		Roots:     roots,
	}, nil
}
