package main

import (
	"github.com/pulumi/pulumi-gcp/sdk/v5/go/gcp/container"
	"github.com/pulumi/pulumi-gcp/sdk/v5/go/gcp/projects"
	"github.com/pulumi/pulumi-gcp/sdk/v5/go/gcp/serviceaccount"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
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

		// Create a GKE standard cluster
		_, err = container.NewCluster(ctx, "goodluck-standard-gke", &container.ClusterArgs{
			Location: pulumi.String("us-central1"),
			Project: pulumi.String(projectId),
			InitialNodeCount: pulumi.Int(1),
			Name: pulumi.String("goodluck-standard-gke"),
			EnableKubernetesAlpha: pulumi.Bool(false),
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
			WorkloadIdentityConfig: &container.ClusterWorkloadIdentityConfigArgs{
				WorkloadPool: pulumi.Sprintf("%s.svc.id.goog", projectId),
			},
		}, pulumi.DependsOn([]pulumi.Resource{service}))
		if err != nil {
			return err
		}

		// kubeconfig := createKubeconfig(cluster.Endpoint, projectId, "goodluck-standard-gke")

		k8sProvider, err := kubernetes.NewProvider(ctx, "gkeProvider", &kubernetes.ProviderArgs{
			Kubeconfig: pulumi.String("~/.kube/config"),
		})
		if err != nil {
			return err
		}

		_, err = v1.NewNamespace(ctx, "service-namespace", &v1.NamespaceArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.StringPtr("goodluck"),
			},
		}, pulumi.Provider(k8sProvider))

		_, err = v1.NewNamespace(ctx, "gsm-namespace", &v1.NamespaceArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.StringPtr("gsm-secrets"),
			},
		}, pulumi.Provider(k8sProvider))

		// Create a GCP service account
		account, err := serviceaccount.NewAccount(ctx, "secretAccessSvcAccount", &serviceaccount.AccountArgs{
			Project:  pulumi.String(projectId),
			AccountId: pulumi.String("secret-accessor-sa"),
			DisplayName: pulumi.String("secret accessor service account"),
			Description: pulumi.String("This service account has the secret accessor role"),
		})
		if err != nil {
			return err
		}
		
		// Assign "Secret Accessor" role to the service account
		_, err = projects.NewIAMMember(ctx, "secretAccessorRole", &projects.IAMMemberArgs{
			Role:    pulumi.String("roles/secretmanager.secretAccessor"),
			Member:  pulumi.Sprintf("serviceAccount:%s", account.Email),
			Project: pulumi.String(projectId),
		})
		if err != nil {
			return err
		}

		ctx.Export("clusterName", pulumi.String("goodluck-standard-gke"))
		ctx.Export("serviceAccountEmail", account.Email)
		
		return nil
	})
}

// Helper function to create a kubeconfig
func createKubeconfig(endpoint pulumi.StringOutput, projectId, clusterName string) pulumi.StringOutput {
	// This will sync the endpoint value from the output and then create a kubeconfig
	return endpoint.ApplyT(func(e string) string {
		return `
apiVersion: v1
clusters:
- cluster:
    server: ` + e + `
  name: ` + clusterName + `
contexts:
- context:
    cluster: ` + clusterName + `
    user: ` + projectId + `
current-context: ` + clusterName + `
kind: Config
preferences: {}
users:
- name: ` + projectId + `
  user:
    auth-provider:
      name: gcp
`
	}).(pulumi.StringOutput)
}