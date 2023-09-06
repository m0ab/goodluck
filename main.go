package main

import (
	"github.com/pulumi/pulumi-gcp/sdk/v5/go/gcp/container"
	"github.com/pulumi/pulumi-gcp/sdk/v5/go/gcp/projects"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		const projectId = "go-gke-gsm-pulumi"
		
		// Enable the required service
		service, err := projects.NewService(ctx, "ContainerEngine", &projects.ServiceArgs{
			Service: pulumi.String("container.googleapis.com"),
			Project: pulumi.String(projectId),
		})
		if err != nil {
			return err
		}

		// Create a GKE Autopilot cluster
		_, err = container.NewCluster(ctx, "goodluck-autopilot-gke", &container.ClusterArgs{
			Location: pulumi.String("us-central1"),
			Project: pulumi.String(projectId),
			InitialNodeCount: pulumi.Int(1),
			Name: pulumi.String("goodluck-autopilot-gke"),
			EnableKubernetesAlpha: pulumi.Bool(false),
			EnableAutopilot: pulumi.Bool(true),
			Network: pulumi.String("default"),
			ReleaseChannel: &container.ClusterReleaseChannelArgs{
				Channel: pulumi.String("REGULAR"),
			},
			MasterAuth: &container.ClusterMasterAuthArgs{
				// Enable client certificate
				ClientCertificateConfig: &container.ClusterMasterAuthClientCertificateConfigArgs{
					IssueClientCertificate: pulumi.Bool(false),
				},
			},
			MasterAuthorizedNetworksConfig: &container.ClusterMasterAuthorizedNetworksConfigArgs{
				CidrBlocks: container.ClusterMasterAuthorizedNetworksConfigCidrBlockArray{
					&container.ClusterMasterAuthorizedNetworksConfigCidrBlockArgs{
						CidrBlock: pulumi.String("0.0.0.0/0"),
					},
				},
			},
		}, pulumi.DependsOn([]pulumi.Resource{service}))
		if err != nil {
			return err
		}

		ctx.Export("clusterName", pulumi.String("goodluck-autopilot-gke"))
		
		return nil
	})
}
