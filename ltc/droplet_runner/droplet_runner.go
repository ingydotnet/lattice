package droplet_runner

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/buildpack_app_lifecycle"
	"github.com/cloudfoundry-incubator/lattice/ltc/app_examiner"
	"github.com/cloudfoundry-incubator/lattice/ltc/app_runner"
	"github.com/cloudfoundry-incubator/lattice/ltc/blob_store"
	"github.com/cloudfoundry-incubator/lattice/ltc/blob_store/blob"
	"github.com/cloudfoundry-incubator/lattice/ltc/config"
	"github.com/cloudfoundry-incubator/lattice/ltc/task_runner"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

const (
	DropletStack         = "cflinuxfs2"
	DropletWindowsStack  = "buildpack"
	DropletRootFS        = "preloaded:" + DropletStack
	DropletWindowsRootFS = "windowsservercore:" + DropletWindowsStack
)

//go:generate counterfeiter -o fake_droplet_runner/fake_droplet_runner.go . DropletRunner
type DropletRunner interface {
	UploadBits(dropletName, uploadPath string) error
	BuildDroplet(taskName, dropletName, buildpackUrl string, environment map[string]string, memoryMB, cpuWeight, diskMB int) error
	BuildWindowsDroplet(taskName, dropletName, buildpackUrl string, environment map[string]string, memoryMB, cpuWeight, diskMB int) error
	LaunchDroplet(appName, dropletName, startCommand string, startArgs []string, appEnvironmentParams app_runner.AppEnvironmentParams) error
	LaunchWindowsDroplet(appName, dropletName, startCommand string, startArgs []string, appEnvironmentParams app_runner.AppEnvironmentParams) error
	ListDroplets() ([]Droplet, error)
	RemoveDroplet(dropletName string) error
	ExportDroplet(dropletName string) (io.ReadCloser, io.ReadCloser, error)
	ImportDroplet(dropletName, dropletPath, metadataPath string) error
}

type Droplet struct {
	Name    string
	Created time.Time
	Size    int64
}

type dropletRunner struct {
	appRunner   app_runner.AppRunner
	taskRunner  task_runner.TaskRunner
	config      *config.Config
	blobStore   BlobStore
	appExaminer app_examiner.AppExaminer
}

//go:generate counterfeiter -o fake_blob_store/fake_blob_store.go . BlobStore
type BlobStore interface {
	List() ([]blob.Blob, error)
	Delete(path string) error
	Upload(path string, contents io.ReadSeeker) error
	Download(path string) (io.ReadCloser, error)

	blob_store.DropletStore
}

type annotation struct {
	DropletSource struct {
		DropletName string `json:"droplet_name"`
	} `json:"droplet_source"`
}

func New(appRunner app_runner.AppRunner, taskRunner task_runner.TaskRunner, config *config.Config, blobStore BlobStore, appExaminer app_examiner.AppExaminer) DropletRunner {
	return &dropletRunner{
		appRunner:   appRunner,
		taskRunner:  taskRunner,
		config:      config,
		blobStore:   blobStore,
		appExaminer: appExaminer,
	}
}

func (dr *dropletRunner) ListDroplets() ([]Droplet, error) {
	blobs, err := dr.blobStore.List()
	if err != nil {
		return nil, err
	}

	droplets := []Droplet{}
	for _, blob := range blobs {
		pathComponents := strings.Split(blob.Path, "/")
		if len(pathComponents) == 2 && (pathComponents[len(pathComponents)-1] == "droplet.tgz" || pathComponents[len(pathComponents)-1] == "droplet.zip") {
			droplets = append(droplets, Droplet{Name: pathComponents[len(pathComponents)-2], Size: blob.Size, Created: blob.Created})
		}
	}

	return droplets, nil
}

func (dr *dropletRunner) UploadBits(dropletName, uploadPath string) error {
	uploadFile, err := os.Open(uploadPath)
	if err != nil {
		return err
	}

	return dr.blobStore.Upload(path.Join(dropletName, "bits.zip"), uploadFile)
}

// TODO: this implementation is hardcoded to use a webdav blobstore
func (dr *dropletRunner) BuildWindowsDroplet(taskName, dropletName, buildpackUrl string, environment map[string]string, memoryMB, cpuWeight, diskMB int) error {

	blobStoreURL := url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%s", dr.config.BlobStore().Host, dr.config.BlobStore().Port),
		User:   url.UserPassword(dr.config.BlobStore().Username, dr.config.BlobStore().Password),
	}

	action := &models.SerialAction{
		Actions: []models.Action{
			&models.DownloadAction{
				From: "https://region-b.geo-1.objects.hpcloudsvc.com/v1/10990308817909/pelerinul/davtool.zip",
				To:   "tmp",
				User: "dummy",
			},
			&models.DownloadAction{
				From: blobStoreURL.String() + "/blobs/" + dropletName + "/bits.zip",
				To:   "tmp\\app",
				User: "dummy",
			},
			&models.DownloadAction{
				From: buildpackUrl,
				To:   "tmp\\buildpack",
				User: "dummy",
			},
			&models.RunAction{
				Path: "c:\\tmp\\davtool",
				Dir:  "c:\\",
				Args: []string{"delete", blobStoreURL.String() + "/blobs/" + dropletName + "/bits.zip"},
				User: "dummy",
			},
			// Detect
			// cmd /c .\bin\detect.bat TestApps\iis8
			// output is "detected_buildpack"
			//
			// Release
			// cmd /c .\bin\release.bat TestApps\iis8
			// output is yaml with "detected_start_command" as "default_process_type/web"
			//
			// Compile
			// cmd /c .\bin\compile.bat TestApps\iis8 cache
			//
			// Sample result.json
			// {
			//  "buildpack_key": "https://github.com/cloudfoundry/nodejs-buildpack",
			//  "detected_buildpack":"",
			//  "execution_metadata":"{\"start_command\":\"npm start\"}",
			//  "detected_start_command":{"web":"npm start"}
			// }
			&models.RunAction{
				Path: "powershell.exe",
				Dir:  "c:\\tmp",
				Args: []string{
					`-command`,
					`"`,
					`cp -Recurse .\buildpack\*\* .\buildpack\ -ErrorAction SilentlyContinue;`,
					`$detectedBuildpack = (& C:\tmp\buildpack\bin\detect.bat C:\tmp\app | Out-String);`,
					`if ($LASTEXITCODE -ne 0) { exit 10 };`,
					`$releaseYaml = (& C:\tmp\buildpack\bin\release.bat C:\tmp\app | Out-String);`,
					`if ($LASTEXITCODE -ne 0) { exit 11 };`,
					`$releaseYaml -match '\s+web:\s(?<start>.+)';`,
					`$startCommand = $Matches['start'].Trim();`,
					`& C:\tmp\buildpack\bin\compile.bat C:\tmp\app c:\tmp\cache;`,
					`if ($LASTEXITCODE -ne 0) { exit 12 };`,
					`$fileSystemAssemblyPath = Join-Path ([System.Runtime.InteropServices.RuntimeEnvironment]::GetRuntimeDirectory()) 'System.IO.Compression.FileSystem.dll';`,
					`Add-Type -Path $fileSystemAssemblyPath;`,
					`[System.IO.Compression.ZipFile]::CreateFromDirectory('c:\tmp\app','c:\tmp\droplet.zip',[System.IO.Compression.CompressionLevel]::Optimal, $false);`,
					`$executionMetadata = (@{ 'start_command' = $startCommand } | ConvertTo-Json | Out-String);`,
					fmt.Sprintf(
						`@{'buildpack-key' = '%s'; 'detected-buildpack' = ''; 'execution_metadata' = $executionMetadata; 'detected_start_command' = @{ 'web' = $startCommand } } | ConvertTo-Json | Out-File -Encoding 'ASCII' c:\tmp\result.json;`,
						buildpackUrl,
					),
					`"`,
				},
				User: "dummy",
			},

			&models.RunAction{
				Path: "c:\\tmp\\davtool",
				Dir:  "c:\\",
				Args: []string{"put", blobStoreURL.String() + "/blobs/" + dropletName + "/droplet.zip", "c:\\tmp\\droplet.zip"},
				User: "dummy",
			},
			&models.RunAction{
				Path: "c:\\tmp\\davtool",
				Dir:  "c:\\",
				Args: []string{"put", blobStoreURL.String() + "/blobs/" + dropletName + "/result.json", "c:\\tmp\\result.json"},
				User: "dummy",
			},
		},
	}

	environment["CF_STACK"] = DropletWindowsStack
	environment["MEMORY_LIMIT"] = fmt.Sprintf("%dM", memoryMB)

	createTaskParams := task_runner.NewCreateTaskParams(
		action,
		taskName,
		DropletWindowsRootFS,
		"lattice",
		"BUILD",
		environment,
		[]models.SecurityGroupRule{},
		memoryMB,
		cpuWeight,
		diskMB,
	)

	return dr.taskRunner.CreateTask(createTaskParams)
}

func (dr *dropletRunner) BuildDroplet(taskName, dropletName, buildpackUrl string, environment map[string]string, memoryMB, cpuWeight, diskMB int) error {
	builderConfig := buildpack_app_lifecycle.NewLifecycleBuilderConfig([]string{buildpackUrl}, true, false)

	action := &models.SerialAction{
		Actions: []models.Action{
			&models.DownloadAction{
				From: "http://file_server.service.dc1.consul:8080/v1/static/lattice-cell-helpers.tgz",
				To:   "/tmp",
				User: "vcap",
			},
			dr.blobStore.DownloadAppBitsAction(dropletName),
			dr.blobStore.DeleteAppBitsAction(dropletName),
			&models.RunAction{
				Path: "/bin/chmod",
				Dir:  "/tmp/app",
				Args: []string{"-R", "a+X", "."},
				User: "vcap",
			},
			&models.RunAction{
				Path: "/tmp/builder",
				Dir:  "/",
				Args: builderConfig.Args(),
				User: "vcap",
			},
			dr.blobStore.UploadDropletAction(dropletName),
			dr.blobStore.UploadDropletMetadataAction(dropletName),
		},
	}

	environment["CF_STACK"] = DropletStack
	environment["MEMORY_LIMIT"] = fmt.Sprintf("%dM", memoryMB)

	createTaskParams := task_runner.NewCreateTaskParams(
		action,
		taskName,
		DropletRootFS,
		"lattice",
		"BUILD",
		environment,
		[]models.SecurityGroupRule{},
		memoryMB,
		cpuWeight,
		diskMB,
	)

	return dr.taskRunner.CreateTask(createTaskParams)
}

func (dr *dropletRunner) LaunchDroplet(appName, dropletName string, startCommand string, startArgs []string, appEnvironmentParams app_runner.AppEnvironmentParams) error {
	executionMetadata, err := dr.getExecutionMetadata(path.Join(dropletName, "result.json"))
	if err != nil {
		return err
	}

	dropletAnnotation := annotation{}
	dropletAnnotation.DropletSource.DropletName = dropletName

	annotationBytes, err := json.Marshal(dropletAnnotation)
	if err != nil {
		return err
	}

	if appEnvironmentParams.EnvironmentVariables == nil {
		appEnvironmentParams.EnvironmentVariables = map[string]string{}
	}

	appEnvironmentParams.EnvironmentVariables["PWD"] = "/home/vcap"
	appEnvironmentParams.EnvironmentVariables["TMPDIR"] = "/home/vcap/tmp"

	appParams := app_runner.CreateAppParams{
		AppEnvironmentParams: appEnvironmentParams,

		Name:         appName,
		RootFS:       DropletRootFS,
		StartCommand: "/tmp/launcher",
		AppArgs: []string{
			"/home/vcap/app",
			strings.Join(append([]string{startCommand}, startArgs...), " "),
			executionMetadata,
		},

		Annotation: string(annotationBytes),

		Setup: &models.SerialAction{
			LogSource: appName,
			Actions: []models.Action{
				&models.DownloadAction{
					From: "http://file_server.service.dc1.consul:8080/v1/static/lattice-cell-helpers.tgz",
					To:   "/tmp",
					User: "vcap",
				},
				&models.DownloadAction{
					From: "http://file_server.service.dc1.consul:8080/v1/static/healthcheck.tgz",
					To:   "/tmp",
					User: "vcap",
				},
				dr.blobStore.DownloadDropletAction(dropletName),
			},
		},
	}

	return dr.appRunner.CreateApp(appParams)
}

func (dr *dropletRunner) LaunchWindowsDroplet(appName, dropletName string, startCommand string, startArgs []string, appEnvironmentParams app_runner.AppEnvironmentParams) error {
	executionMetadata, err := dr.getExecutionMetadata(path.Join(dropletName, "result.json"))
	if err != nil {
		return err
	}

	blobStoreURL := url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%s", dr.config.BlobStore().Host, dr.config.BlobStore().Port),
		User:   url.UserPassword(dr.config.BlobStore().Username, dr.config.BlobStore().Password),
	}

	if appEnvironmentParams.EnvironmentVariables == nil {
		appEnvironmentParams.EnvironmentVariables = map[string]string{}
	}

	appEnvironmentParams.EnvironmentVariables["HOME"] = "c:\\app"
	appEnvironmentParams.EnvironmentVariables["HOMEPATH"] = "c:\\"

	appEnvironmentParams.WorkingDir = "c:\\app"
	appEnvironmentParams.Monitor = app_runner.MonitorConfig{
		Method: app_runner.WindowsMonitor,
	}

	var executionMetadataJSON interface{}
	err = json.Unmarshal([]byte(executionMetadata), &executionMetadataJSON)
	if err != nil {
		return err
	}
	executionMetadataMap := executionMetadataJSON.(map[string]interface{})

	appParams := app_runner.CreateAppParams{
		AppEnvironmentParams: appEnvironmentParams,

		Name:         appName,
		RootFS:       DropletWindowsRootFS,
		StartCommand: "powershell.exe",

		AppArgs: []string{
			`-command`,
			`"`,
			`cd c:\app;`,
			fmt.Sprintf(`& %s;`, executionMetadataMap["start_command"]),
			`"`,
		},

		Setup: &models.SerialAction{
			LogSource: appName,
			Actions: []models.Action{
				&models.DownloadAction{
					From: blobStoreURL.String() + "/blobs/" + dropletName + "/droplet.zip",
					To:   "app",
					User: "vcap",
				},
			},
		},
	}

	return dr.appRunner.CreateApp(appParams)
}

func (dr *dropletRunner) getExecutionMetadata(path string) (string, error) {
	reader, err := dr.blobStore.Download(path)
	if err != nil {
		return "", err
	}

	var result struct {
		ExecutionMetadata string `json:"execution_metadata"`
	}

	if err := json.NewDecoder(reader).Decode(&result); err != nil {
		return "", err
	}

	return result.ExecutionMetadata, nil
}

func dropletMatchesAnnotation(dropletName string, a annotation) bool {
	return a.DropletSource.DropletName == dropletName
}

func (dr *dropletRunner) RemoveDroplet(dropletName string) error {
	apps, err := dr.appExaminer.ListApps()
	if err != nil {
		return err
	}
	for _, app := range apps {
		dropletAnnotation := annotation{}
		if err := json.Unmarshal([]byte(app.Annotation), &dropletAnnotation); err != nil {
			continue
		}

		if dropletMatchesAnnotation(dropletName, dropletAnnotation) {
			return fmt.Errorf("app %s was launched from droplet", app.ProcessGuid)
		}
	}

	blobs, err := dr.blobStore.List()
	if err != nil {
		return err
	}

	found := false
	for _, blob := range blobs {
		if strings.HasPrefix(blob.Path, dropletName+"/") {
			if err := dr.blobStore.Delete(blob.Path); err != nil {
				return err
			} else {
				found = true
			}
		}
	}

	if !found {
		return errors.New("droplet not found")
	}

	return nil
}

func (dr *dropletRunner) ExportDroplet(dropletName string) (io.ReadCloser, io.ReadCloser, error) {
	dropletReader, err := dr.blobStore.Download(path.Join(dropletName, "droplet.tgz"))
	if err != nil {
		return nil, nil, fmt.Errorf("droplet not found: %s", err)
	}

	metadataReader, err := dr.blobStore.Download(path.Join(dropletName, "result.json"))
	if err != nil {
		return nil, nil, fmt.Errorf("metadata not found: %s", err)
	}

	return dropletReader, metadataReader, err
}

func (dr *dropletRunner) ImportDroplet(dropletName, dropletPath, metadataPath string) error {
	dropletFile, err := os.Open(dropletPath)
	if err != nil {
		return err
	}
	metadataFile, err := os.Open(metadataPath)
	if err != nil {
		return err
	}

	if err := dr.blobStore.Upload(path.Join(dropletName, "droplet.tgz"), dropletFile); err != nil {
		return err
	}

	if err := dr.blobStore.Upload(path.Join(dropletName, "result.json"), metadataFile); err != nil {
		return err
	}

	return nil
}
