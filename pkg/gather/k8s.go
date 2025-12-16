package gather

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/gruntwork-io/terratest/modules/testing"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/require"
)

// K8sAfterAll provides cleanup and data collection logic for Kubernetes resources.
// It collects crust-gather information, archives it, and adds it to the report.
func K8sAfterAll(ctx context.Context, t testing.TestingT, resourceWaitTimeout time.Duration) {
	timeBoundContext, cancel := context.WithTimeout(ctx, resourceWaitTimeout)
	defer cancel()

	// Collect crust-gather folder
	cmd := exec.CommandContext(timeBoundContext, "kubectl-crust-gather", "collect", "-f", "../crust-gather")
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	err := cmd.Run()
	if err != nil {
		logger.Default.Logf(t, "crust-gather collect failed: %v, stdout: %s, stderr: %s", err, outb.String(), errb.String())
		require.NoError(t, err, "crust-gather collect failed")
	} else {
		if errb.Len() > 0 {
			logger.Default.Logf(t, "crust-gather collect stderr: %s", errb.String())
		}
	}

	// Archive crust-gather folder
	tarGzFileName := "/tmp/crust-gather.tar.gz"
	cmd = exec.CommandContext(timeBoundContext, "tar", "-czvf", tarGzFileName, "../crust-gather")
	outb.Reset()
	errb.Reset()
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	err = cmd.Run()
	if err != nil {
		logger.Default.Logf(t, "tar command failed: %v, stdout: %s, stderr: %s", err, outb.String(), errb.String())
		require.NoError(t, err, "tar command failed")
	} else {
		if errb.Len() > 0 {
			logger.Default.Logf(t, "tar command stderr: %s", errb.String())
		}
	}

	// Add crust-gather.tar.gz to report
	tarGzFileContent, err := os.ReadFile(tarGzFileName)
	if err != nil {
		logger.Default.Logf(t, "failed to read %s: %v", tarGzFileName, err)
		require.NoError(t, err, fmt.Sprintf("failed to read %s", tarGzFileName))
	} else {
		baseName := filepath.Base(tarGzFileName)
		logger.Default.Logf(t, "Saved crust-gather.tar.gz to %s", tarGzFileName)
		ginkgo.AddReportEntry(baseName, string(tarGzFileContent), ginkgo.ReportEntryVisibilityNever)
	}
}
