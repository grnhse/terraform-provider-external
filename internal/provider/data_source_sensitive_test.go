package provider

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const testDatasourceSensitiveConfig_basic = `
data "external" "test" {
  program = ["%s", "cheese"]

  query = {
    value = "pizza"
  }
}

output "query_value" {
  value = "${data.external.test.result["query_value"]}"
}

output "argument" {
  value = "${data.external.test.result["argument"]}"
}
`

func TestDatasourceSensitive_basic(t *testing.T) {
	programPath, err := buildDatasourceSensitiveTestProgram()
	if err != nil {
		t.Fatal(err)
		return
	}

	resource.UnitTest(t, resource.TestCase{
		Providers: testProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testDatasourceSensitiveConfig_basic, programPath),
				Check: func(s *terraform.State) error {
					_, ok := s.RootModule().Resources["data.external.test"]
					if !ok {
						return fmt.Errorf("missing data resource")
					}

					outputs := s.RootModule().Outputs

					if outputs["argument"] == nil {
						return fmt.Errorf("missing 'argument' output")
					}
					if outputs["query_value"] == nil {
						return fmt.Errorf("missing 'query_value' output")
					}

					if outputs["argument"].Value != "cheese" {
						return fmt.Errorf(
							"'argument' output is %q; want 'cheese'",
							outputs["argument"].Value,
						)
					}
					if outputs["query_value"].Value != "pizza" {
						return fmt.Errorf(
							"'query_value' output is %q; want 'pizza'",
							outputs["query_value"].Value,
						)
					}

					return nil
				},
			},
		},
	})
}

const testDatasourceSensitiveConfig_error = `
data "external" "test" {
  program = ["%s"]

  query = {
    fail = "true"
  }
}
`

func TestDatasourceSensitive_error(t *testing.T) {
	programPath, err := buildDatasourceSensitiveTestProgram()
	if err != nil {
		t.Fatal(err)
		return
	}

	resource.UnitTest(t, resource.TestCase{
		Providers: testProviders,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testDatasourceSensitiveConfig_error, programPath),
				ExpectError: regexp.MustCompile("I was asked to fail"),
			},
		},
	})
}

func buildDatasourceSensitiveTestProgram() (string, error) {
	// We have a simple Go program that we use as a stub for testing.
	cmd := exec.Command(
		"go", "install",
		"github.com/terraform-providers/terraform-provider-external/internal/provider/test-programs/tf-acc-external-data-source",
	)
	err := cmd.Run()

	if err != nil {
		return "", fmt.Errorf("failed to build test stub program: %s", err)
	}

	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = filepath.Join(os.Getenv("HOME") + "/go")
	}

	programPath := path.Join(
		filepath.SplitList(gopath)[0], "bin", "tf-acc-external-data-source",
	)
	return programPath, nil
}
