package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/alecthomas/chroma/quick"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var k8sCmd = &cobra.Command{
	Use:   "k8s",
	Short: "Display Kubernetes cluster information",
	Long: `Display detailed Kubernetes cluster information using k8s.io/client-go.
Provides information about:
  - Node status and resources
  - Pod status and resource usage
  - Namespace information
  - K3s specific configuration`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := log.FromContext(cmd.Context())

		for {
			if err := showK8sInfo(cmd.Context(), logger); err != nil {
				return err
			}

			if !watchOutput {
				break
			}
			time.Sleep(2 * time.Second)
		}
		return nil
	},
}

func showK8sInfo(ctx context.Context, logger *log.Logger) error {
	logger.Debug("gathering kubernetes information")

	// Use the default kubeconfig location for k3s
	kubeconfig := filepath.Join("/etc", "rancher", "k3s", "k3s.yaml")
	if _, err := os.Stat(kubeconfig); os.IsNotExist(err) {
		// Fallback to default location
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	info := make(map[string]interface{})

	// Get nodes information
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to get nodes: %w", err)
	}
	info["nodes"] = nodes.Items

	// Get namespaces
	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to get namespaces: %w", err)
	}
	info["namespaces"] = namespaces.Items

	// Get pods from all namespaces
	pods, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to get pods: %w", err)
	}
	info["pods"] = pods.Items

	var b []byte

	if outputJSON {
		b, err = json.MarshalIndent(info, "", "  ")
	} else {
		b, err = yaml.Marshal(info)
	}
	if err != nil {
		return fmt.Errorf("failed to marshal kubernetes info: %w", err)
	}

	if rawOutput {
		fmt.Println(string(b))
		return nil
	}

	style := "catppuccin-latte"
	if lipgloss.HasDarkBackground() {
		style = "catppuccin-frappe"
	}

	format := "yaml"
	if outputJSON {
		format = "json"
	}

	return quick.Highlight(os.Stdout, string(b), format, "terminal256", style)
}

func init() {
	rootCmd.AddCommand(k8sCmd)
}
