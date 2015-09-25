package models

import (
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v2"

	envmanModels "github.com/bitrise-io/envman/models"
	"github.com/bitrise-io/go-utils/pointers"
	stepmanModels "github.com/bitrise-io/stepman/models"
	"github.com/stretchr/testify/require"
)

// ----------------------------
// --- Validate

// Workflow
func TestValidate(t *testing.T) {
	workflow := WorkflowModel{
		BeforeRun: []string{"befor1", "befor2", "befor3"},
		AfterRun:  []string{"after1", "after2", "after3"},
	}
	require.Equal(t, nil, workflow.Validate())

	configStr := `
format_version: 1.0.0
default_step_lib_source: "https://github.com/bitrise-io/bitrise-steplib.git"

workflows:
  target:
    envs:
    - ENV_KEY: env_value
      opts:
        title: test_env
    title: Output Test
    steps:
    - script:
        title: Should fail
        inputs:
        - content: echo "Hello"
          BAD_KEY: value
`

	config := BitriseDataModel{}
	require.Equal(t, nil, yaml.Unmarshal([]byte(configStr), &config))
	require.Equal(t, nil, config.Normalize())
	require.NotEqual(t, nil, config.Validate())
	require.Equal(t, true, strings.Contains(config.Validate().Error(), "Invalid env: more than 2 fields:"))
}

// ----------------------------
// --- Merge

func TestMergeEnvironmentWith(t *testing.T) {
	// Different keys
	diffEnv := envmanModels.EnvironmentItemModel{
		"test_key": "test_value",
		envmanModels.OptionsKey: envmanModels.EnvironmentItemOptionsModel{
			Title:             pointers.NewStringPtr("test_title"),
			Description:       pointers.NewStringPtr("test_description"),
			Summary:           pointers.NewStringPtr("test_summary"),
			ValueOptions:      []string{"test_valu_options1", "test_valu_options2"},
			IsRequired:        pointers.NewBoolPtr(true),
			IsExpand:          pointers.NewBoolPtr(false),
			IsDontChangeValue: pointers.NewBoolPtr(true),
		},
	}
	env := envmanModels.EnvironmentItemModel{
		"test_key1": "test_value",
	}
	require.NotEqual(t, nil, MergeEnvironmentWith(&env, diffEnv))

	// Normal merge
	env = envmanModels.EnvironmentItemModel{
		"test_key":              "test_value",
		envmanModels.OptionsKey: envmanModels.EnvironmentItemOptionsModel{},
	}
	require.Equal(t, nil, MergeEnvironmentWith(&env, diffEnv))

	options, err := env.GetOptions()
	require.Equal(t, nil, err)

	diffOptions, err := diffEnv.GetOptions()
	require.Equal(t, nil, err)

	require.Equal(t, *diffOptions.Title, *options.Title)
	require.Equal(t, *diffOptions.Description, *options.Description)
	require.Equal(t, *diffOptions.Summary, *options.Summary)
	require.Equal(t, len(diffOptions.ValueOptions), len(options.ValueOptions))
	require.Equal(t, *diffOptions.IsRequired, *options.IsRequired)
	require.Equal(t, *diffOptions.IsExpand, *options.IsExpand)
	require.Equal(t, *diffOptions.IsDontChangeValue, *options.IsDontChangeValue)
}

func TestMergeStepWith(t *testing.T) {
	desc := "desc 1"
	summ := "sum 1"
	website := "web/1"
	fork := "fork/1"
	published := time.Date(2012, time.January, 1, 0, 0, 0, 0, time.UTC)

	stepData := stepmanModels.StepModel{
		Description:         pointers.NewStringPtr(desc),
		Summary:             pointers.NewStringPtr(summ),
		Website:             pointers.NewStringPtr(website),
		SourceCodeURL:       pointers.NewStringPtr(fork),
		PublishedAt:         pointers.NewTimePtr(published),
		HostOsTags:          []string{"osx"},
		ProjectTypeTags:     []string{"ios"},
		TypeTags:            []string{"test"},
		IsRequiresAdminUser: pointers.NewBoolPtr(true),
		Inputs: []envmanModels.EnvironmentItemModel{
			envmanModels.EnvironmentItemModel{
				"KEY_1": "Value 1",
			},
			envmanModels.EnvironmentItemModel{
				"KEY_2": "Value 2",
			},
		},
		Outputs: []envmanModels.EnvironmentItemModel{},
	}

	diffTitle := "name 2"
	newSuppURL := "supp"
	runIfStr := ""
	stepDiffToMerge := stepmanModels.StepModel{
		Title:      pointers.NewStringPtr(diffTitle),
		HostOsTags: []string{"linux"},
		Source: stepmanModels.StepSourceModel{
			Git: "https://git.url",
		},
		Dependencies: []stepmanModels.DependencyModel{
			stepmanModels.DependencyModel{
				Manager: "brew",
				Name:    "test",
			},
		},
		SupportURL: pointers.NewStringPtr(newSuppURL),
		RunIf:      pointers.NewStringPtr(runIfStr),
		Inputs: []envmanModels.EnvironmentItemModel{
			envmanModels.EnvironmentItemModel{
				"KEY_2": "Value 2 CHANGED",
			},
		},
	}

	mergedStepData, err := MergeStepWith(stepData, stepDiffToMerge)
	require.Equal(t, nil, err)

	t.Logf("-> MERGED Step Data: %#v\n", mergedStepData)

	require.Equal(t, "name 2", *mergedStepData.Title)
	require.Equal(t, "desc 1", *mergedStepData.Description)
	require.Equal(t, "sum 1", *mergedStepData.Summary)
	require.Equal(t, "web/1", *mergedStepData.Website)
	require.Equal(t, "fork/1", *mergedStepData.SourceCodeURL)
	require.Equal(t, true, (*mergedStepData.PublishedAt).Equal(time.Date(2012, time.January, 1, 0, 0, 0, 0, time.UTC)))
	require.Equal(t, "linux", mergedStepData.HostOsTags[0])
	require.Equal(t, "", *mergedStepData.RunIf)
	require.Equal(t, 1, len(mergedStepData.Dependencies))

	dep := mergedStepData.Dependencies[0]
	require.Equal(t, "brew", dep.Manager)
	require.Equal(t, "test", dep.Name)

	// inputs
	input0 := mergedStepData.Inputs[0]
	key0, value0, err := input0.GetKeyValuePair()

	require.Equal(t, nil, err)
	require.Equal(t, "KEY_1", key0)
	require.Equal(t, "Value 1", value0)

	input1 := mergedStepData.Inputs[1]
	key1, value1, err := input1.GetKeyValuePair()

	require.Equal(t, nil, err)
	require.Equal(t, "KEY_2", key1)
	require.Equal(t, "Value 2 CHANGED", value1)
}

func TestGetInputByKey(t *testing.T) {
	stepData := stepmanModels.StepModel{
		Inputs: []envmanModels.EnvironmentItemModel{
			envmanModels.EnvironmentItemModel{
				"KEY_1": "Value 1",
			},
			envmanModels.EnvironmentItemModel{
				"KEY_2": "Value 2",
			},
		},
	}

	_, found := getInputByKey(stepData, "KEY_1")
	require.Equal(t, true, found)

	_, found = getInputByKey(stepData, "KEY_3")
	require.Equal(t, false, found)
}

// ----------------------------
// --- StepIDData

func TestGetStepIDStepDataPair(t *testing.T) {
	stepData := stepmanModels.StepModel{}

	stepListItem := StepListItemModel{
		"step1": stepData,
	}

	id, _, err := GetStepIDStepDataPair(stepListItem)
	require.Equal(t, nil, err)
	require.Equal(t, "step1", id)

	stepListItem = StepListItemModel{
		"step1": stepData,
		"step2": stepData,
	}

	id, _, err = GetStepIDStepDataPair(stepListItem)
	require.NotEqual(t, nil, err)
}

func TestCreateStepIDDataFromString(t *testing.T) {
	// default / long / verbose ID mode
	stepCompositeIDString := "steplib-src::step-id@0.0.1"
	t.Log("stepCompositeIDString: ", stepCompositeIDString)
	stepIDData, err := CreateStepIDDataFromString(stepCompositeIDString, "")
	if err != nil {
		t.Fatal("Failed to create StepIDData from composite-id: ", stepCompositeIDString, "| err:", err)
	}
	t.Logf("stepIDData:%#v", stepIDData)
	if stepIDData.SteplibSource != "steplib-src" {
		t.Fatal("stepIDData.SteplibSource incorrectly converted:", stepIDData.SteplibSource)
	}
	if stepIDData.IDorURI != "step-id" {
		t.Fatal("stepIDData.IDorURI incorrectly converted:", stepIDData.IDorURI)
	}
	if stepIDData.Version != "0.0.1" {
		t.Fatal("stepIDData.Version incorrectly converted:", stepIDData.Version)
	}

	// no steplib-source
	stepCompositeIDString = "step-id@0.0.1"
	t.Log("(no steplib-source, but default provided) stepCompositeIDString: ", stepCompositeIDString)
	stepIDData, err = CreateStepIDDataFromString(stepCompositeIDString, "default-steplib-src")
	if err != nil {
		t.Fatal("Failed to create StepIDData from composite-id: ", stepCompositeIDString, "| err:", err)
	}
	t.Logf("stepIDData:%#v", stepIDData)
	if stepIDData.SteplibSource != "default-steplib-src" {
		t.Fatal("stepIDData.SteplibSource incorrectly converted:", stepIDData.SteplibSource)
	}
	if stepIDData.IDorURI != "step-id" {
		t.Fatal("stepIDData.IDorURI incorrectly converted:", stepIDData.IDorURI)
	}
	if stepIDData.Version != "0.0.1" {
		t.Fatal("stepIDData.Version incorrectly converted:", stepIDData.Version)
	}

	// invalid/empty step lib source, but default provided
	stepCompositeIDString = "::step-id@0.0.1"
	t.Log("(invalid/empty steplib source, but default provided) stepCompositeIDString: ", stepCompositeIDString)
	stepIDData, err = CreateStepIDDataFromString(stepCompositeIDString, "default-steplib-src")
	if err != nil {
		t.Fatal("Failed to create StepIDData from composite-id: ", stepCompositeIDString, "| err:", err)
	}
	t.Logf("stepIDData:%#v", stepIDData)
	if stepIDData.SteplibSource != "default-steplib-src" {
		t.Fatal("stepIDData.SteplibSource incorrectly converted:", stepIDData.SteplibSource)
	}
	if stepIDData.IDorURI != "step-id" {
		t.Fatal("stepIDData.IDorURI incorrectly converted:", stepIDData.IDorURI)
	}
	if stepIDData.Version != "0.0.1" {
		t.Fatal("stepIDData.Version incorrectly converted:", stepIDData.Version)
	}

	// invalid/empty step lib source + no default
	stepCompositeIDString = "::step-id@0.0.1"
	t.Log("(invalid/empty steplib source) stepCompositeIDString: ", stepCompositeIDString)
	stepIDData, err = CreateStepIDDataFromString(stepCompositeIDString, "")
	if err == nil {
		t.Fatal("Should fail to parse the ID if it contains an empty steplib-src and no default src is provided")
	}

	// no steplib-source & no default -> fail
	stepCompositeIDString = "step-id@0.0.1"
	t.Log("(no steplib-source & no default, should fail) stepCompositeIDString: ", stepCompositeIDString)
	stepIDData, err = CreateStepIDDataFromString(stepCompositeIDString, "")
	if err == nil {
		t.Fatal("Should fail to parse the ID if it does not contain a steplib-src and no default src is provided")
	} else {
		t.Log("Expected error (ok): ", err)
	}

	// no steplib & no version, only step-id
	stepCompositeIDString = "step-id"
	t.Log("no steplib & no version, only step-id and default lib source: ", stepCompositeIDString)
	stepIDData, err = CreateStepIDDataFromString(stepCompositeIDString, "def-lib-src")
	if err != nil {
		t.Fatal("Failed to create StepIDData from composite-id: ", stepCompositeIDString, "| err:", err)
	}
	t.Logf("stepIDData:%#v", stepIDData)
	if stepIDData.SteplibSource != "def-lib-src" {
		t.Fatal("stepIDData.SteplibSource incorrectly converted:", stepIDData.SteplibSource)
	}
	if stepIDData.IDorURI != "step-id" {
		t.Fatal("stepIDData.IDorURI incorrectly converted:", stepIDData.IDorURI)
	}
	if stepIDData.Version != "" {
		t.Fatal("stepIDData.Version incorrectly converted:", stepIDData.Version)
	}

	// empty test
	stepCompositeIDString = ""
	t.Log("Empty stepCompositeIDString test")
	stepIDData, err = CreateStepIDDataFromString(stepCompositeIDString, "def-step-src")
	if err == nil {
		t.Fatal("Should fail to parse the ID from an empty string! (at least the step-id is required)")
	} else {
		t.Log("Expected error (ok): ", err)
	}

	// special empty test
	stepCompositeIDString = "@1.0.0"
	t.Log("Empty stepCompositeIDString test with only version")
	stepIDData, err = CreateStepIDDataFromString(stepCompositeIDString, "def-step-src")
	if err == nil {
		t.Fatal("Should fail to parse the ID from an empty string! (at least the step-id is required)")
	} else {
		t.Log("Expected error (ok): ", err)
	}

	//
	// ----- Local Path
	stepCompositeIDString = "path::/some/path"
	t.Log("LOCAL - stepCompositeIDString: ", stepCompositeIDString)
	stepIDData, err = CreateStepIDDataFromString(stepCompositeIDString, "")
	if err != nil {
		t.Fatal("Failed to create StepIDData from composite-id: ", stepCompositeIDString, "| err:", err)
	}
	t.Logf("stepIDData:%#v", stepIDData)
	if stepIDData.SteplibSource != "path" {
		t.Fatal("stepIDData.SteplibSource incorrectly converted:", stepIDData.SteplibSource)
	}
	if stepIDData.IDorURI != "/some/path" {
		t.Fatal("stepIDData.IDorURI incorrectly converted:", stepIDData.IDorURI)
	}
	if stepIDData.Version != "" {
		t.Fatal("stepIDData.Version incorrectly converted:", stepIDData.Version)
	}

	stepCompositeIDString = "path::~/some/path/in/home"
	t.Log("LOCAL - path should be preserved as-it-is, #1 - stepCompositeIDString: ", stepCompositeIDString)
	stepIDData, err = CreateStepIDDataFromString(stepCompositeIDString, "")
	if err != nil {
		t.Fatal("Failed to create StepIDData from composite-id: ", stepCompositeIDString, "| err:", err)
	}
	t.Logf("stepIDData:%#v", stepIDData)
	if stepIDData.SteplibSource != "path" {
		t.Fatal("stepIDData.SteplibSource incorrectly converted:", stepIDData.SteplibSource)
	}
	if stepIDData.IDorURI != "~/some/path/in/home" {
		t.Fatal("stepIDData.IDorURI incorrectly converted:", stepIDData.IDorURI)
	}
	if stepIDData.Version != "" {
		t.Fatal("stepIDData.Version incorrectly converted:", stepIDData.Version)
	}

	stepCompositeIDString = "path::$HOME/some/path/in/home"
	t.Log("LOCAL - path should be preserved as-it-is, #1 - stepCompositeIDString: ", stepCompositeIDString)
	stepIDData, err = CreateStepIDDataFromString(stepCompositeIDString, "")
	if err != nil {
		t.Fatal("Failed to create StepIDData from composite-id: ", stepCompositeIDString, "| err:", err)
	}
	t.Logf("stepIDData:%#v", stepIDData)
	if stepIDData.SteplibSource != "path" {
		t.Fatal("stepIDData.SteplibSource incorrectly converted:", stepIDData.SteplibSource)
	}
	if stepIDData.IDorURI != "$HOME/some/path/in/home" {
		t.Fatal("stepIDData.IDorURI incorrectly converted:", stepIDData.IDorURI)
	}
	if stepIDData.Version != "" {
		t.Fatal("stepIDData.Version incorrectly converted:", stepIDData.Version)
	}

	//
	// ----- Direct git uri
	stepCompositeIDString = "git::https://github.com/bitrise-io/steps-timestamp.git@develop"
	t.Log("DIRECT-GIT - http(s) - stepCompositeIDString: ", stepCompositeIDString)
	stepIDData, err = CreateStepIDDataFromString(stepCompositeIDString, "some-def-coll")
	if err != nil {
		t.Fatal("Failed to create StepIDData from composite-id: ", stepCompositeIDString, "| err:", err)
	}
	t.Logf("stepIDData:%#v", stepIDData)
	if stepIDData.SteplibSource != "git" {
		t.Fatal("stepIDData.SteplibSource incorrectly converted:", stepIDData.SteplibSource)
	}
	if stepIDData.IDorURI != "https://github.com/bitrise-io/steps-timestamp.git" {
		t.Fatal("stepIDData.IDorURI incorrectly converted:", stepIDData.IDorURI)
	}
	if stepIDData.Version != "develop" {
		t.Fatal("stepIDData.Version incorrectly converted:", stepIDData.Version)
	}

	stepCompositeIDString = "git::git@github.com:bitrise-io/steps-timestamp.git@develop"
	t.Log("DIRECT-GIT - ssh - stepCompositeIDString: ", stepCompositeIDString)
	stepIDData, err = CreateStepIDDataFromString(stepCompositeIDString, "")
	if err != nil {
		t.Fatal("Failed to create StepIDData from composite-id: ", stepCompositeIDString, "| err:", err)
	}
	t.Logf("stepIDData:%#v", stepIDData)
	if stepIDData.SteplibSource != "git" {
		t.Fatal("stepIDData.SteplibSource incorrectly converted:", stepIDData.SteplibSource)
	}
	if stepIDData.IDorURI != "git@github.com:bitrise-io/steps-timestamp.git" {
		t.Fatal("stepIDData.IDorURI incorrectly converted:", stepIDData.IDorURI)
	}
	if stepIDData.Version != "develop" {
		t.Fatal("stepIDData.Version incorrectly converted:", stepIDData.Version)
	}

	//
	// ----- Old step
	stepCompositeIDString = "_::https://github.com/bitrise-io/steps-timestamp.git@1.0.0"
	t.Log("OLD-STEP - stepCompositeIDString: ", stepCompositeIDString)
	stepIDData, err = CreateStepIDDataFromString(stepCompositeIDString, "")
	if err != nil {
		t.Fatal("Failed to create StepIDData from composite-id: ", stepCompositeIDString, "| err:", err)
	}
	t.Logf("stepIDData:%#v", stepIDData)
	if stepIDData.SteplibSource != "_" {
		t.Fatal("stepIDData.SteplibSource incorrectly converted:", stepIDData.SteplibSource)
	}
	if stepIDData.IDorURI != "https://github.com/bitrise-io/steps-timestamp.git" {
		t.Fatal("stepIDData.IDorURI incorrectly converted:", stepIDData.IDorURI)
	}
	if stepIDData.Version != "1.0.0" {
		t.Fatal("stepIDData.Version incorrectly converted:", stepIDData.Version)
	}
}

// ----------------------------
// --- RemoveRedundantFields

func TestRemoveEnvironmentRedundantFields(t *testing.T) {
	// Trivial remove - all fields should be default value
	env := envmanModels.EnvironmentItemModel{
		"TEST_KEY": "test_value",
		envmanModels.OptionsKey: envmanModels.EnvironmentItemOptionsModel{
			Title:             pointers.NewStringPtr(""),
			Description:       pointers.NewStringPtr(""),
			Summary:           pointers.NewStringPtr(""),
			ValueOptions:      []string{},
			IsRequired:        pointers.NewBoolPtr(envmanModels.DefaultIsRequired),
			IsExpand:          pointers.NewBoolPtr(envmanModels.DefaultIsExpand),
			IsDontChangeValue: pointers.NewBoolPtr(envmanModels.DefaultIsDontChangeValue),
		},
	}

	if err := removeEnvironmentRedundantFields(&env); err != nil {
		t.Fatal("Failed to remove redundant fields:", err)
	}

	options, err := env.GetOptions()
	if err != nil {
		t.Fatal("Failed to get env options:", err)
	}

	if options.Title != nil {
		t.Fatal("options.Title should be nil")
	}
	if options.Description != nil {
		t.Fatal("options.Description should be nil")
	}
	if options.Summary != nil {
		t.Fatal("options.Summary should be nil")
	}
	if len(options.ValueOptions) != 0 {
		t.Fatal("options.ValueOptions should be empty")
	}
	if options.IsRequired != nil {
		t.Fatal("options.IsRequired should be nil")
	}
	if options.IsExpand != nil {
		t.Fatal("options.IsExpand should be nil")
	}
	if options.IsDontChangeValue != nil {
		t.Fatal("options.IsDontChangeValue should be nil")
	}

	// Trivial don't remove - no fields should be default value
	env = envmanModels.EnvironmentItemModel{
		"TEST_KEY": "test_value",
		envmanModels.OptionsKey: envmanModels.EnvironmentItemOptionsModel{
			Title:             pointers.NewStringPtr("t"),
			Description:       pointers.NewStringPtr("d"),
			Summary:           pointers.NewStringPtr("s"),
			ValueOptions:      []string{"i"},
			IsRequired:        pointers.NewBoolPtr(true),
			IsExpand:          pointers.NewBoolPtr(false),
			IsDontChangeValue: pointers.NewBoolPtr(true),
		},
	}

	if err := removeEnvironmentRedundantFields(&env); err != nil {
		t.Fatal("Failed to remove redundant fields:", err)
	}

	options, err = env.GetOptions()
	if err != nil {
		t.Fatal("Failed to get env options:", err)
	}

	if *options.Title != "t" {
		t.Fatal("options.Title should be: t")
	}
	if *options.Description != "d" {
		t.Fatal("options.Description should be: d")
	}
	if *options.Summary != "s" {
		t.Fatal("options.Summary should be: s")
	}
	if options.ValueOptions[0] != "i" {
		t.Fatal("options.ValueOptions should be: {i}")
	}
	if *options.IsRequired != true {
		t.Fatal("options.IsRequired should be: false")
	}
	if *options.IsExpand != false {
		t.Fatal("options.IsExpand should be: false")
	}
	if *options.IsDontChangeValue != true {
		t.Fatal("options.IsDontChangeValue should be: true")
	}

	// No options - opts field shouldn't exist
	env = envmanModels.EnvironmentItemModel{
		"TEST_KEY": "test_value",
	}

	if err := removeEnvironmentRedundantFields(&env); err != nil {
		t.Fatal("Failed to remove redundant fields:", err)
	}

	_, ok := env[envmanModels.OptionsKey]
	if ok {
		t.Fatal("opts field shouldn't exist")
	}
}

func configModelFromYAMLBytes(configBytes []byte) (bitriseData BitriseDataModel, err error) {
	if err = yaml.Unmarshal(configBytes, &bitriseData); err != nil {
		return
	}
	return
}

func TestRemoveWorkflowRedundantFields(t *testing.T) {
	configStr := `
format_version: 1.0.0
default_step_lib_source: "https://github.com/bitrise-io/bitrise-steplib.git"

app:
  summary: "sum"
  envs:
  - ENV_KEY: env_value
    opts:
      is_required: true

workflows:
  target:
    envs:
    - ENV_KEY: env_value
      opts:
        title: test_env
    title: Output Test
    steps:
    - script:
        description: test
`

	config, err := configModelFromYAMLBytes([]byte(configStr))
	if err != nil {
		t.Fatal(err)
	}
	if err := config.RemoveRedundantFields(); err != nil {
		t.Fatal("Failed to remove redundant fields:", err)
	}

	if config.App.Title != "" {
		t.Fatal("config.App.Title should be empty")
	}
	if config.App.Description != "" {
		t.Fatal("config.App.Description should be empty")
	}
	if config.App.Summary != "sum" {
		t.Fatal("config.App.Summary should be: sum")
	}
	for _, env := range config.App.Environments {
		options, err := env.GetOptions()
		if err != nil {
			t.Fatal("Failed to get env options:", err)
		}

		if options.Title != nil {
			t.Fatal("options.Title should be nil")
		}
		if options.Description != nil {
			t.Fatal("options.Description should be nil")
		}
		if options.Summary != nil {
			t.Fatal("options.Summary should be nil")
		}
		if len(options.ValueOptions) != 0 {
			t.Fatal("options.ValueOptions should be empty")
		}
		if *options.IsRequired != true {
			t.Fatal("options.IsRequired should be: true")
		}
		if options.IsExpand != nil {
			t.Fatal("options.IsExpand should be nil")
		}
		if options.IsDontChangeValue != nil {
			t.Fatal("options.IsDontChangeValue should be nil")
		}
	}

	for _, workflow := range config.Workflows {
		if workflow.Title != "Output Test" {
			t.Fatal("workflow.Title should be: Output Test")
		}
		if workflow.Description != "" {
			t.Fatal("workflow.Description should be empty")
		}
		if workflow.Summary != "" {
			t.Fatal("workflow.Summary should be empty")
		}

		for _, env := range workflow.Environments {
			options, err := env.GetOptions()
			if err != nil {
				t.Fatal("Failed to get env options:", err)
			}

			if *options.Title != "test_env" {
				t.Fatal("options.Title should be: test_env")
			}
			if options.Description != nil {
				t.Fatal("options.Description should be nil")
			}
			if options.Summary != nil {
				t.Fatal("options.Summary should be nil")
			}
			if len(options.ValueOptions) != 0 {
				t.Fatal("options.ValueOptions should be empty")
			}
			if options.IsRequired != nil {
				t.Fatal("options.IsRequired should be: false")
			}
			if options.IsExpand != nil {
				t.Fatal("options.IsExpand should be nil")
			}
			if options.IsDontChangeValue != nil {
				t.Fatal("options.IsDontChangeValue should be nil")
			}
		}

		for _, stepListItem := range workflow.Steps {
			_, step, err := GetStepIDStepDataPair(stepListItem)
			if err != nil {
				t.Fatal("Faild to get step id data:", err)
			}
			if step.Title != nil {
				t.Fatal("step.Title should be nil")
			}
			if *step.Description != "test" {
				t.Fatal("step.Description should be: test")
			}
			if step.Summary != nil {
				t.Fatal("step.Summary should be nil")
			}
			if step.Website != nil {
				t.Fatal("step.Website should be nil")
			}
			if step.SourceCodeURL != nil {
				t.Fatal("step.SourceCodeURL should be nil")
			}
			if step.SupportURL != nil {
				t.Fatal("step.SupportURL should be nil")
			}
			if step.PublishedAt != nil {
				t.Fatal("step.PublishedAt should be nil")
			}
			if step.Source.Git != "" || step.Source.Commit != "" {
				t.Fatal("step.Source.Git && step.Source.Commit should be empty")
			}
			if len(step.HostOsTags) != 0 {
				t.Fatal("len(step.HostOsTags) should be 0")
			}
			if len(step.ProjectTypeTags) != 0 {
				t.Fatal("len(step.ProjectTypeTags) should be 0")
			}
			if len(step.TypeTags) != 0 {
				t.Fatal("len(step.TypeTags) should be 0")
			}
			if step.IsRequiresAdminUser != nil {
				t.Fatal("step.IsRequiresAdminUser should be nil")
			}
			if step.IsAlwaysRun != nil {
				t.Fatal("step.IsAlwaysRun should be nil")
			}
			if step.IsSkippable != nil {
				t.Fatal("step.IsSkippable should be nil")
			}
			if step.RunIf != nil {
				t.Fatal("step.RunIf should be nil")
			}

			if len(step.Inputs) != 0 {
				t.Fatal("len(step.Inputs) should be 0")
			}
			if len(step.Outputs) != 0 {
				t.Fatal("len(step.Outputs) should be 0")
			}
		}
	}
}
