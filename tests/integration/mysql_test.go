// +build integration

package integration

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/bitly/go-simplejson"
	log "github.com/sirupsen/logrus"

	"github.com/newrelic/nri-mysql/tests/integration/helpers"
	"github.com/newrelic/nri-mysql/tests/integration/jsonschema"
)

var (
	iName = "mysql"

	defaultContainer = "integration_nri-mysql_1"
	// mysql config
	defaultBinPath   = "/nr-mysql"
	defaultMysqlUser = "dbuser"
	defaultMysqlPass = "DBpwd1234!"
	defaultMysqlHost = "mysql"
	defaultMysqlPort = 3306

	// cli flags
	container = flag.String("container", defaultContainer, "container where the integration is installed")
	update    = flag.Bool("test.update", false, "update json-schema file")
	binPath   = flag.String("bin", defaultBinPath, "Integration binary path")
	user      = flag.String("user", defaultMysqlUser, "Mysql user name")
	psw       = flag.String("psw", defaultMysqlPass, "Mysql user password")
	host      = flag.String("host", defaultMysqlHost, "Mysql host ip address")
	port      = flag.Int("port", defaultMysqlPort, "Mysql port")
)

// Returns the standard output, or fails testing if the command returned an error
func runIntegration(t *testing.T) string {
	t.Helper()

	stdout, stderr, err := helpers.ExecInContainer(*container, []string{*binPath,
		"--username", *user, "--password", *psw, "--hostname", *host, "--port", fmt.Sprint(*port)})

	log.Debug("Integration command Standard Error: ", stderr)
	assert
}



func setup() error {
	flag.Parse()

	if testing.Verbose() {
		log.SetLevel(log.DebugLevel)
	}

	return nil
}

func teardown() error {
	return nil
}

func TestMain(m *testing.M) {
	err := setup()
	if err != nil {
		fmt.Println(err)
		tErr := teardown()
		if tErr != nil {
			fmt.Printf("Error during the teardown of the tests: %s\n", tErr)
		}
		os.Exit(1)
	}

	result := m.Run()

	err = teardown()
	if err != nil {
		fmt.Printf("Error during the teardown of the tests: %s\n", err)
	}

	os.Exit(result)
}

func TestOutputIsValidJSON(t *testing.T) {
	stdout, _, err :=
	if err != nil {
		t.Fatal(err)
	}

	var j map[string]interface{}
	err = json.Unmarshal([]byte(stdout), &j)
	if err != nil {
		t.Error("Integration output should be a JSON dict")
	}
}

func TestMySQLIntegrationValidArguments(t *testing.T) {
	testName := helpers.GetTestName(t)
	cmd := exec.Command(*binPath)
	cmd.Env = []string{
		fmt.Sprintf("USERNAME=%s", *user),
		fmt.Sprintf("PASSWORD=%s", *psw),
		fmt.Sprintf("HOSTNAME=%s", *host),
		fmt.Sprintf("PORT=%d", *port),
		fmt.Sprintf("NRIA_CACHE_PATH=%v", testName),
	}

	var outbuf, errbuf bytes.Buffer
	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf
	err := cmd.Run()
	if err != nil {
		t.Fatalf("It isn't possible to execute MySQL integration binary. Err: %s -- %s", err, errbuf.String())
	}

	schemaPath := filepath.Join("json-schema-files", "mysql-schema-master.json")
	if *update {
		schema, err := jsonschema.Generate(outbuf.String())
		if err != nil {
			t.Fatal(err)
		}

		schemaJSON, err := simplejson.NewJson(schema)
		if err != nil {
			t.Fatalf("Cannot unmarshal JSON schema, got error: %v", err)
		}
		err = helpers.ModifyJSONSchemaGlobal(schemaJSON, iName, 1, "1.1.0")
		if err != nil {
			t.Fatal(err)
		}
		err = helpers.ModifyJSONSchemaInventoryPresent(schemaJSON)
		if err != nil {
			t.Fatal(err)
		}
		err = helpers.ModifyJSONSchemaMetricsPresent(schemaJSON, "MysqlSample")
		if err != nil {
			t.Fatal(err)
		}
		schema, err = schemaJSON.MarshalJSON()
		if err != nil {
			t.Fatalf("Cannot marshal JSON schema, got error: %v", err)
		}
		err = ioutil.WriteFile(schemaPath, schema, 0644)
		if err != nil {
			t.Fatal(err)
		}
	}

	err = jsonschema.Validate(schemaPath, outbuf.String())
	if err != nil {
		t.Fatalf("The output of MySQL integration doesn't have expected format. Err: %s", err)
	}
}

func TestMySQLIntegrationOnlyMetrics(t *testing.T) {
	testName := helpers.GetTestName(t)
	cmd := exec.Command(*binPath)
	cmd.Env = []string{
		fmt.Sprintf("USERNAME=%s", *user),
		fmt.Sprintf("PASSWORD=%s", *psw),
		fmt.Sprintf("HOSTNAME=%s", *host),
		fmt.Sprintf("PORT=%d", *port),
		"METRICS=true",
		fmt.Sprintf("NRIA_CACHE_PATH=%v", testName),
	}

	var outbuf, errbuf bytes.Buffer
	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf
	err := cmd.Run()
	if err != nil {
		t.Fatalf("It isn't possible to execute MySQL integration binary. Err: %s -- %s", err, errbuf.String())
	}

	schemaPath := filepath.Join("json-schema-files", "mysql-schema-metrics-master.json")
	if *update {
		schema, err := jsonschema.Generate(outbuf.String())
		if err != nil {
			t.Fatal(err)
		}

		schemaJSON, err := simplejson.NewJson(schema)
		if err != nil {
			t.Fatalf("Cannot unmarshal JSON schema, got error: %v", err)
		}
		err = helpers.ModifyJSONSchemaGlobal(schemaJSON, iName, 1, "1.1.0")
		if err != nil {
			t.Fatal(err)
		}
		err = helpers.ModifyJSONSchemaNoInventory(schemaJSON)
		if err != nil {
			t.Fatal(err)
		}
		err = helpers.ModifyJSONSchemaMetricsPresent(schemaJSON, "MysqlSample")
		if err != nil {
			t.Fatal(err)
		}
		schema, err = schemaJSON.MarshalJSON()
		if err != nil {
			t.Fatalf("Cannot marshal JSON schema, got error: %v", err)
		}
		err = ioutil.WriteFile(schemaPath, schema, 0644)
		if err != nil {
			t.Fatal(err)
		}
	}

	err = jsonschema.Validate(schemaPath, outbuf.String())
	if err != nil {
		t.Fatalf("The output of MySQL integration doesn't have expected format. Err: %s", err)
	}
}

func TestMySQLIntegrationOnlyInventory(t *testing.T) {
	t.Skip("Skipping test - fix in the MySQL integration required")
	testName := helpers.GetTestName(t)
	cmd := exec.Command(*binPath)
	cmd.Env = []string{
		fmt.Sprintf("USERNAME=%s", *user),
		fmt.Sprintf("PASSWORD=%s", *psw),
		fmt.Sprintf("HOSTNAME=%s", *host),
		fmt.Sprintf("PORT=%d", *port),
		"INVENTORY=true",
		fmt.Sprintf("NRIA_CACHE_PATH=%v", testName),
	}

	var outbuf, errbuf bytes.Buffer
	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf
	err := cmd.Run()
	if err != nil {
		t.Fatalf("It isn't possible to execute MySQL integration binary. Err: %s -- %s", err, errbuf.String())
	}

	schemaPath := filepath.Join("json-schema-files", "mysql-schema-inventory-master.json")
	if *update {
		schema, err := jsonschema.Generate(outbuf.String())
		if err != nil {
			t.Fatal(err)
		}

		schemaJSON, err := simplejson.NewJson(schema)
		if err != nil {
			t.Fatalf("Cannot unmarshal JSON schema, got error: %v", err)
		}
		err = helpers.ModifyJSONSchemaGlobal(schemaJSON, iName, 1, "1.1.0")
		if err != nil {
			t.Fatal(err)
		}
		err = helpers.ModifyJSONSchemaInventoryPresent(schemaJSON)
		if err != nil {
			t.Fatal(err)
		}
		err = helpers.ModifyJSONSchemaNoMetrics(schemaJSON)
		if err != nil {
			t.Fatal(err)
		}
		schema, err = schemaJSON.MarshalJSON()
		if err != nil {
			t.Fatalf("Cannot marshal JSON schema, got error: %v", err)
		}
		err = ioutil.WriteFile(schemaPath, schema, 0644)
		if err != nil {
			t.Fatal(err)
		}
	}

	err = jsonschema.Validate(schemaPath, outbuf.String())
	if err != nil {
		t.Fatalf("The output of MySQL integration doesn't have expected format. Err: %s", err)
	}
}

func TestMySQLIntegrationErrorNoPassword(t *testing.T) {
	testName := helpers.GetTestName(t)
	cmd := exec.Command(*binPath)

	cmd.Env = []string{
		fmt.Sprintf("USERNAME=%s", *user),
		//fmt.Sprintf("PASSWORD=%s", *psw),
		fmt.Sprintf("HOSTNAME=%s", *host),
		fmt.Sprintf("PORT=%d", *port),
		fmt.Sprintf("NRIA_CACHE_PATH=%v", testName),
	}
	expectedErrorMessage := "Access denied "
	var outbuf, errbuf bytes.Buffer
	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf
	err := cmd.Run()
	if err == nil {
		t.Fatal("Error not returned")
	}
	errMatch, _ := regexp.MatchString(expectedErrorMessage, errbuf.String())
	if !errMatch {
		t.Fatalf("Expected error message: '%s', got: '%s'", expectedErrorMessage, errbuf.String())
	}
	if outbuf.String() != "" {
		t.Fatalf("Unexpected output: %s", outbuf.String())
	}
}

func TestMySQLIntegrationErrorWrongPassword(t *testing.T) {
	testName := helpers.GetTestName(t)
	cmd := exec.Command(*binPath)

	cmd.Env = []string{
		fmt.Sprintf("USERNAME=%s", *user),
		fmt.Sprintf("HOSTNAME=%s", *host),
		fmt.Sprintf("PORT=%d", *port),
		"PASSWORD=wrong_password",
		fmt.Sprintf("NRIA_CACHE_PATH=%v", testName),
	}
	expectedErrorMessage := "Access denied "
	var outbuf, errbuf bytes.Buffer
	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf
	err := cmd.Run()
	if err == nil {
		t.Fatal("Error not returned")
	}
	errMatch, _ := regexp.MatchString(expectedErrorMessage, errbuf.String())
	if !errMatch {
		t.Fatalf("Expected error message: '%s', got: '%s'", expectedErrorMessage, errbuf.String())
	}
	if outbuf.String() != "" {
		t.Fatalf("Unexpected output: %s", outbuf.String())
	}
}

func TestMySQLIntegrationErrorNoUsername(t *testing.T) {
	cmd := exec.Command("/bin/sh", "-c", "mysql --version")
	version, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Cannot get MySQL version, got err: %v", err)
	}

	testName := helpers.GetTestName(t)
	cmd = exec.Command(*binPath)
	cmd.Env = []string{
		//fmt.Sprintf("USERNAME=%s", user),
		fmt.Sprintf("PASSWORD=%s", *psw),
		fmt.Sprintf("HOSTNAME=%s", *host),
		fmt.Sprintf("PORT=%d", *port),
		fmt.Sprintf("NRIA_CACHE_PATH=%v", testName),
	}
	var outbuf, errbuf bytes.Buffer
	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf
	err = cmd.Run()

	re := regexp.MustCompile(`Distrib (5\.\d\.)(\d+)`)
	matches := re.FindStringSubmatch(string(version))

	switch {
	case matches != nil && matches[1] == "5.6.":
		t.Logf("MySQL version: %s%s", matches[1], matches[2])
		if err != nil {
			t.Fatalf("It isn't possible to execute MySQL integration binary. Err: %s -- %s", err, errbuf.String())
		}
		schemaPath := filepath.Join("json-schema-files", "mysql-schema-without-replication-grant-master.json")
		if *update {
			schema, err := jsonschema.Generate(outbuf.String())
			if err != nil {
				t.Fatal(err)
			}

			schemaJSON, err := simplejson.NewJson(schema)
			if err != nil {
				t.Fatalf("Cannot unmarshal JSON schema, got error: %v", err)
			}
			err = helpers.ModifyJSONSchemaGlobal(schemaJSON, iName, 1, "1.1.0")
			if err != nil {
				t.Fatal(err)
			}
			err = helpers.ModifyJSONSchemaInventoryPresent(schemaJSON)
			if err != nil {
				t.Fatal(err)
			}
			err = helpers.ModifyJSONSchemaMetricsPresent(schemaJSON, "MysqlSample")
			if err != nil {
				t.Fatal(err)
			}
			schema, err = schemaJSON.MarshalJSON()
			if err != nil {
				t.Fatalf("Cannot marshal JSON schema, got error: %v", err)
			}
			err = ioutil.WriteFile(schemaPath, schema, 0644)
			if err != nil {
				t.Fatal(err)
			}
		}

		err = jsonschema.Validate(schemaPath, outbuf.String())
		if err != nil {
			t.Fatalf("The output of MySQL integration doesn't have expected format. Err: %s", err)
		}

	case matches != nil && matches[1] == "5.7.":
		t.Logf("MySQL version: %s%s", matches[1], matches[2])
		expectedErrorMessage := "Access denied "
		errMatch, _ := regexp.MatchString(expectedErrorMessage, errbuf.String())
		if err == nil {
			t.Fatal("Error not returned")
		}
		if !errMatch {
			t.Fatalf("Expected error message: '%s', got: '%s'", expectedErrorMessage, errbuf.String())
		}
		if outbuf.String() != "" {
			t.Fatalf("Unexpected output: %s", outbuf.String())
		}
	case matches == nil:
		t.Fatal("MySQL version doesn't match against regular expression, version not retrieved")
	default:
		t.Fatalf("MySQL version not as expected, got: %s, expected version: 5.6 or 5.7", matches[1])
	}
}

func TestMySQLIntegrationWrongHostname(t *testing.T) {
	testName := helpers.GetTestName(t)
	cmd := exec.Command(*binPath)

	cmd.Env = []string{
		fmt.Sprintf("USERNAME=%s", *user),
		fmt.Sprintf("PASSWORD=%s", *psw),
		fmt.Sprintf("PORT=%d", *port),
		"HOSTNAME=nonExistingHost",
		fmt.Sprintf("NRIA_CACHE_PATH=%v", testName),
	}
	expectedErrorMessage := "no such host"
	var outbuf, errbuf bytes.Buffer
	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf
	err := cmd.Run()
	if err == nil {
		t.Fatal("Error not returned")
	}
	errMatch, _ := regexp.MatchString(expectedErrorMessage, errbuf.String())
	if !errMatch {
		t.Fatalf("Expected error message: '%s', got: '%s'", expectedErrorMessage, errbuf.String())
	}
	if outbuf.String() != "" {
		t.Fatalf("Unexpected output: %s", outbuf.String())
	}
}

func TestMySQLIntegrationWrongPort(t *testing.T) {
	testName := helpers.GetTestName(t)
	cmd := exec.Command(*binPath)

	cmd.Env = []string{
		fmt.Sprintf("USERNAME=%s", *user),
		fmt.Sprintf("PASSWORD=%s", *psw),
		fmt.Sprintf("HOSTNAME=%s", *host),
		"PORT=1",
		fmt.Sprintf("NRIA_CACHE_PATH=%v", testName),
	}
	expectedErrorMessage := "connection refused"
	var outbuf, errbuf bytes.Buffer
	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf
	err := cmd.Run()
	if err == nil {
		t.Fatal("Error not returned")
	}
	errMatch, _ := regexp.MatchString(expectedErrorMessage, errbuf.String())
	if !errMatch {
		t.Fatalf("Expected error message: '%s', got: '%s'", expectedErrorMessage, errbuf.String())
	}
	if outbuf.String() != "" {
		t.Fatalf("Unexpected output: %s", outbuf.String())
	}
}
