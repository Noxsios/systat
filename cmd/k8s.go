package cmd

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var k8sCmd = &cobra.Command{
	Use:   "k8s",
	Short: "Display Kubernetes cluster information",
	Long: `Display detailed information about your Kubernetes cluster.
Provides information about:
  - Nodes and their status
  - Namespaces and resource usage
  - Pods and their state
  - Services and endpoints`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := log.FromContext(cmd.Context())
		return showK8sInfo(logger)
	},
}

func showK8sInfo(logger *log.Logger) error {
	logger.Debug("gathering kubernetes information")

	// Build kubeconfig path
	home := homedir.HomeDir()
	if home == "" {
		return fmt.Errorf("could not find home directory")
	}
	kubeconfig := filepath.Join(home, ".kube", "config")

	// Load kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create clientset: %w", err)
	}

	if rawOutput {
		return showRawK8sInfo(clientset)
	}

	// Get nodes
	nodes, err := clientset.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to get nodes: %w", err)
	}

	fmt.Println(titleStyle.Render("Kubernetes Nodes"))
	columns := []table.Column{
		{Title: "Name", Width: 30},
		{Title: "Status", Width: 10},
		{Title: "Version", Width: 15},
		{Title: "OS", Width: 15},
		{Title: "Kernel", Width: 20},
	}

	var rows []table.Row
	for _, node := range nodes.Items {
		rows = append(rows, table.Row{
			node.Name,
			string(node.Status.Phase),
			node.Status.NodeInfo.KubeletVersion,
			node.Status.NodeInfo.OperatingSystem,
			node.Status.NodeInfo.KernelVersion,
		})
	}

	t := NewTable(columns, rows)
	fmt.Println(tableStyle.Render(t.View()))

	// Get namespaces
	namespaces, err := clientset.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to get namespaces: %w", err)
	}

	fmt.Println(titleStyle.Render("Kubernetes Namespaces"))
	columns = []table.Column{
		{Title: "Name", Width: 30},
		{Title: "Status", Width: 10},
		{Title: "Age", Width: 15},
	}

	rows = nil
	for _, ns := range namespaces.Items {
		rows = append(rows, table.Row{
			ns.Name,
			string(ns.Status.Phase),
			ns.CreationTimestamp.String(),
		})
	}

	t = NewTable(columns, rows)
	fmt.Println(tableStyle.Render(t.View()))

	return nil
}

func showRawK8sInfo(clientset *kubernetes.Clientset) error {
	// Get nodes
	nodes, err := clientset.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to get nodes: %w", err)
	}

	fmt.Println("Kubernetes Nodes:")
	for _, node := range nodes.Items {
		fmt.Printf("  Name: %s\n", node.Name)
		fmt.Printf("    Status: %s\n", node.Status.Phase)
		fmt.Printf("    Version: %s\n", node.Status.NodeInfo.KubeletVersion)
		fmt.Printf("    OS: %s\n", node.Status.NodeInfo.OperatingSystem)
		fmt.Printf("    Kernel: %s\n", node.Status.NodeInfo.KernelVersion)
		fmt.Println()
	}

	// Get namespaces
	namespaces, err := clientset.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to get namespaces: %w", err)
	}

	fmt.Println("Kubernetes Namespaces:")
	for _, ns := range namespaces.Items {
		fmt.Printf("  Name: %s\n", ns.Name)
		fmt.Printf("    Status: %s\n", ns.Status.Phase)
		fmt.Printf("    Age: %s\n", ns.CreationTimestamp.String())
		fmt.Println()
	}

	return nil
}

func init() {
	rootCmd.AddCommand(k8sCmd)
}
