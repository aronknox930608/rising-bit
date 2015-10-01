package bitrise

import (
	"fmt"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/bitrise-io/bitrise/models"
	"github.com/bitrise-io/go-utils/colorstring"
	"github.com/bitrise-io/go-utils/stringutil"
	"github.com/bitrise-io/go-utils/versions"
	stepmanModels "github.com/bitrise-io/stepman/models"
)

const (
	// should not be under ~45
	stepRunSummaryBoxWidthInChars = 80
)

// IsUpdateAvailable ...
func IsUpdateAvailable(stepInfo stepmanModels.StepInfoModel) bool {
	if stepInfo.Latest == "" {
		return false
	}

	res, err := versions.CompareVersions(stepInfo.Version, stepInfo.Latest)
	if err != nil {
		log.Debugf("Failed to compare versions, err: %s", err)
	}

	return (res == 1)
}

// PrintRunningWorkflow ...
func PrintRunningWorkflow(title string) {
	fmt.Println()
	log.Info(colorstring.Bluef("Running workflow (%s)", title))
	fmt.Println()
}

func getRunningStepHeaderMainSection(stepInfo stepmanModels.StepInfoModel, idx int) string {
	title := stepInfo.Title
	version := stepInfo.Version

	if len(version) > 25 {
		version = "..." + stringutil.MaxLastChars(version, 22)
	}
	content := fmt.Sprintf("| (%d) %s (%s) |", idx, title, version)
	charDiff := len(content) - stepRunSummaryBoxWidthInChars

	if charDiff < 0 {
		// shorter than desired - fill with space
		content = fmt.Sprintf("| (%d) %s (%s)%s |", idx, title, version, strings.Repeat(" ", -charDiff))
	} else if charDiff > 0 {
		// longer than desired - trim title
		trimmedTitleWidth := len(title) - charDiff - 3
		if trimmedTitleWidth < 0 {
			log.Errorf("Step Version too long, can't present title at all! : %s", version)
		} else {
			content = fmt.Sprintf("| (%d) %s... (%s) |", idx, title[0:trimmedTitleWidth], version)
		}
	}
	return content
}

func getRunningStepHeaderSubSection(stepInfo stepmanModels.StepInfoModel) string {
	id := stepInfo.ID
	version := stepInfo.Version
	collection := stepInfo.StepLib
	logTime := time.Now().Format(time.RFC3339)

	idRow := fmt.Sprintf("| id: %s |", id)
	charDiff := len(idRow) - stepRunSummaryBoxWidthInChars
	if charDiff < 0 {
		// shorter than desired - fill with space
		idRow = fmt.Sprintf("| id: %s%s |", id, strings.Repeat(" ", -charDiff))
	} else if charDiff > 0 {
		// longer than desired - trim title
		trimmedWidth := len(id) - charDiff - 3
		if trimmedWidth < 0 {
			log.Errorf("Step id too long, can't present id at all! : %s", id)
		} else {
			idRow = fmt.Sprintf("| id: %s... |", id[:trimmedWidth])
		}
	}

	versionRow := fmt.Sprintf("| version: %s |", version)
	charDiff = len(versionRow) - stepRunSummaryBoxWidthInChars
	if charDiff < 0 {
		// shorter than desired - fill with space
		versionRow = fmt.Sprintf("| version: %s%s |", version, strings.Repeat(" ", -charDiff))
	} else if charDiff > 0 {
		// longer than desired - trim title
		trimmedWidth := len(version) - charDiff - 3
		if trimmedWidth < 0 {
			log.Errorf("Step version too long, can't present version at all! : %s", version)
		} else {
			versionRow = fmt.Sprintf("| id: %s... |", version[:trimmedWidth])
		}
	}

	collectionRow := fmt.Sprintf("| collection: %s |", collection)
	charDiff = len(collectionRow) - stepRunSummaryBoxWidthInChars
	if charDiff < 0 {
		// shorter than desired - fill with space
		collectionRow = fmt.Sprintf("| collection: %s%s |", collection, strings.Repeat(" ", -charDiff))
	} else if charDiff > 0 {
		// longer than desired - trim title
		trimmedWidth := len(collection) - charDiff - 3
		if trimmedWidth < 0 {
			log.Errorf("Step collection too long, can't present collection at all! : %s", version)
		} else {
			collectionRow = fmt.Sprintf("| collection: ...%s |", collection[len(collection)-trimmedWidth:])
		}
	}

	timeRow := fmt.Sprintf("| time: %s |", logTime)
	charDiff = len(timeRow) - stepRunSummaryBoxWidthInChars
	if charDiff < 0 {
		// shorter than desired - fill with space
		timeRow = fmt.Sprintf("| time: %s%s |", logTime, strings.Repeat(" ", -charDiff))
	} else if charDiff > 0 {
		// longer than desired - trim title
		trimmedWidth := len(logTime) - charDiff - 3
		if trimmedWidth < 0 {
			log.Errorf("Time too long, can't present time at all! : %s", version)
		} else {
			timeRow = fmt.Sprintf("| time: %s... |", logTime[:trimmedWidth])
		}
	}

	return fmt.Sprintf("%s\n%s\n%s\n%s", idRow, versionRow, collectionRow, timeRow)
}

// PrintRunningStepHeader ...
func PrintRunningStepHeader(stepInfo stepmanModels.StepInfoModel, idx int) {
	sep := fmt.Sprintf("+%s+", strings.Repeat("-", stepRunSummaryBoxWidthInChars-2))

	fmt.Println(sep)
	fmt.Println(getRunningStepHeaderMainSection(stepInfo, idx))
	fmt.Println(sep)
	fmt.Println(getRunningStepHeaderSubSection(stepInfo))
	fmt.Println(sep)
	fmt.Println("|" + strings.Repeat(" ", stepRunSummaryBoxWidthInChars-2) + "|")
}

func getTrimmedStepName(stepRunResult models.StepRunResultsModel) string {
	iconBoxWidth := len("    ")
	timeBoxWidth := len(" time (s) ")
	titleBoxWidth := stepRunSummaryBoxWidthInChars - 4 - iconBoxWidth - timeBoxWidth - 1

	stepInfo := stepRunResult.StepInfo

	title := stepInfo.ID
	version := stepInfo.Version
	if len(version) > 25 {
		version = "..." + stringutil.MaxLastChars(version, 22)
	}
	titleBox := ""
	switch stepRunResult.Status {
	case models.StepRunStatusCodeSuccess, models.StepRunStatusCodeSkipped, models.StepRunStatusCodeSkippedWithRunIf:
		titleBox = fmt.Sprintf("%s (%s)", title, version)
		if len(titleBox) > titleBoxWidth {
			dif := len(titleBox) - titleBoxWidth
			title = title[:len(title)-dif-3] + "..."
			titleBox = fmt.Sprintf("%s (%s)", title, version)
		}
		break
	case models.StepRunStatusCodeFailed, models.StepRunStatusCodeFailedSkippable:
		titleBox = fmt.Sprintf("%s (%s) (exit code: %d)", title, version, stepRunResult.ExitCode)
		if len(titleBox) > titleBoxWidth {
			dif := len(titleBox) - titleBoxWidth
			title = title[:len(title)-dif-3] + "..."
			titleBox = fmt.Sprintf("%s (%s) (exit code: %d)", title, version, stepRunResult.ExitCode)
		}
		break
	default:
		log.Error("Unkown result code")
		return ""
	}
	return titleBox
}

func getRunningStepFooterMainSection(stepRunResult models.StepRunResultsModel) string {
	iconBoxWidth := len("    ")
	timeBoxWidth := len(" time (s) ")
	titleBoxWidth := stepRunSummaryBoxWidthInChars - 4 - iconBoxWidth - timeBoxWidth - 1

	icon := ""
	title := getTrimmedStepName(stepRunResult)
	runTimeStr := TimeToFormattedSeconds(stepRunResult.RunTime, " sec")
	coloringFunc := colorstring.Green
	switch stepRunResult.Status {
	case models.StepRunStatusCodeSuccess:
		icon = "✅"
		coloringFunc = colorstring.Green
		break
	case models.StepRunStatusCodeFailed:
		icon = "🚫"
		coloringFunc = colorstring.Red
		break
	case models.StepRunStatusCodeFailedSkippable:
		icon = "⚠️"
		coloringFunc = colorstring.Yellow
		break
	case models.StepRunStatusCodeSkipped, models.StepRunStatusCodeSkippedWithRunIf:
		icon = "➡"
		coloringFunc = colorstring.Blue
		break
	default:
		log.Error("Unkown result code")
		return ""
	}

	iconBox := fmt.Sprintf(" %s  ", icon)

	titleWhiteSpaceWidth := titleBoxWidth - len(title)
	titleBox := fmt.Sprintf(" %s%s", coloringFunc(title), strings.Repeat(" ", titleWhiteSpaceWidth))

	timeWhiteSpaceWidth := timeBoxWidth - len(runTimeStr) - 1
	timeBox := fmt.Sprintf(" %s%s", runTimeStr, strings.Repeat(" ", timeWhiteSpaceWidth))

	return fmt.Sprintf("|%s|%s|%s|", iconBox, titleBox, timeBox)
}

func getRunningStepFooterSubSection(stepRunResult models.StepRunResultsModel, isUpdateAvailable bool) string {
	stepInfo := stepRunResult.StepInfo

	updateRow := ""
	if isUpdateAvailable {
		updateRow = fmt.Sprintf("| Update available: %s -> %s |", stepInfo.Version, stepInfo.Latest)
		charDiff := len(updateRow) - stepRunSummaryBoxWidthInChars
		if charDiff < 0 {
			// shorter than desired - fill with space
			updateRow = fmt.Sprintf("| Update available: %s -> %s%s |", stepInfo.Version, stepInfo.Latest, strings.Repeat(" ", -charDiff))
		} else if charDiff > 0 {
			// longer than desired - trim title
			if charDiff > 6 {
				updateRow = fmt.Sprintf("| Update available!%s |", strings.Repeat(" ", -len("| Update available! |")-stepRunSummaryBoxWidthInChars))
			} else {
				updateRow = fmt.Sprintf("| Update available: -> %s%s |", stepInfo.Latest, strings.Repeat(" ", -len("| Update available: -> %s |")-stepRunSummaryBoxWidthInChars))
			}
		}
	}

	issueRow := fmt.Sprintf("| Issue tracker: %s |", stepInfo.SupportURL)
	if stepInfo.SupportURL != "" {
		charDiff := len(issueRow) - stepRunSummaryBoxWidthInChars
		if charDiff < 0 {
			// shorter than desired - fill with space
			issueRow = fmt.Sprintf("| Issue tracker: %s%s |", stepInfo.SupportURL, strings.Repeat(" ", -charDiff))
		} else if charDiff > 0 {
			// longer than desired - trim title
			trimmedWidth := len(stepInfo.SupportURL) - charDiff - 3
			if trimmedWidth < 0 {
				log.Errorf("Support url too long, can't present support url at all! : %s", stepInfo.SupportURL)
			} else {
				issueRow = fmt.Sprintf("| Issue tracker: ...%s |", stepInfo.SupportURL[len(stepInfo.SupportURL)-trimmedWidth:])
			}
		}
	}

	sourceRow := fmt.Sprintf("| Source: %s |", stepInfo.SourceCodeURL)
	if stepInfo.SourceCodeURL != "" {
		charDiff := len(sourceRow) - stepRunSummaryBoxWidthInChars
		if charDiff < 0 {
			// shorter than desired - fill with space
			sourceRow = fmt.Sprintf("| Source: %s%s |", stepInfo.SourceCodeURL, strings.Repeat(" ", -charDiff))
		} else if charDiff > 0 {
			// longer than desired - trim title
			trimmedWidth := len(stepInfo.SourceCodeURL) - charDiff - 3
			if trimmedWidth < 0 {
				log.Errorf("Source url too long, can't present source url at all! : %s", stepInfo.SourceCodeURL)
			} else {
				sourceRow = fmt.Sprintf("| Source: ...%s |", stepInfo.SourceCodeURL[len(stepInfo.SourceCodeURL)-trimmedWidth:])
			}
		}
	}

	content := ""
	if isUpdateAvailable {
		content = fmt.Sprintf("%s", updateRow)
	}
	if stepInfo.SupportURL != "" {
		if content != "" {
			content = fmt.Sprintf("%s\n%s", content, issueRow)
		} else {
			content = fmt.Sprintf("%s", issueRow)
		}
	}
	if stepInfo.SourceCodeURL != "" {
		if content != "" {
			content = fmt.Sprintf("%s\n%s", content, sourceRow)
		} else {
			content = fmt.Sprintf("%s", sourceRow)
		}
	}
	return content
}

// PrintRunningStepFooter ..
func PrintRunningStepFooter(stepRunResult models.StepRunResultsModel, isLastStepInWorkflow bool) {
	iconBoxWidth := len("    ")
	timeBoxWidth := len(" time (s) ")
	titleBoxWidth := stepRunSummaryBoxWidthInChars - 4 - iconBoxWidth - timeBoxWidth
	sep := fmt.Sprintf("+%s+%s+%s+", strings.Repeat("-", iconBoxWidth), strings.Repeat("-", titleBoxWidth), strings.Repeat("-", timeBoxWidth))

	fmt.Println("|" + strings.Repeat(" ", stepRunSummaryBoxWidthInChars-2) + "|")

	fmt.Println(sep)
	fmt.Println(getRunningStepFooterMainSection(stepRunResult))
	fmt.Println(sep)
	if stepRunResult.Error != nil {
		isUpdateAvailable := IsUpdateAvailable(stepRunResult.StepInfo)
		fmt.Println(getRunningStepFooterSubSection(stepRunResult, isUpdateAvailable))
		fmt.Println(sep)
	}

	if !isLastStepInWorkflow {
		fmt.Println()
		fmt.Println(strings.Repeat(" ", 42) + "▼")
		fmt.Println()
	}
}

// PrintSummary ...
func PrintSummary(buildRunResults models.BuildRunResultsModel) {
	iconBoxWidth := len("    ")
	timeBoxWidth := len(" time (s) ")
	titleBoxWidth := stepRunSummaryBoxWidthInChars - 4 - iconBoxWidth - timeBoxWidth

	fmt.Println()
	fmt.Println()
	fmt.Printf("+%s+\n", strings.Repeat("-", stepRunSummaryBoxWidthInChars-2))
	whitespaceWidth := (stepRunSummaryBoxWidthInChars - 2 - len("bitrise summary")) / 2
	fmt.Printf("|%sbitrise summary%s|\n", strings.Repeat(" ", whitespaceWidth), strings.Repeat(" ", whitespaceWidth))
	fmt.Printf("+%s+%s+%s+\n", strings.Repeat("-", iconBoxWidth), strings.Repeat("-", titleBoxWidth), strings.Repeat("-", timeBoxWidth))

	whitespaceWidth = stepRunSummaryBoxWidthInChars - len("|    | title") - len("| time (s) |")
	fmt.Printf("|    | title%s| time (s) |\n", strings.Repeat(" ", whitespaceWidth))
	fmt.Printf("+%s+%s+%s+\n", strings.Repeat("-", iconBoxWidth), strings.Repeat("-", titleBoxWidth), strings.Repeat("-", timeBoxWidth))

	orderedResults := buildRunResults.OrderedResults()
	tmpTime := time.Time{}
	for _, stepRunResult := range orderedResults {
		tmpTime = tmpTime.Add(stepRunResult.RunTime)
		fmt.Println(getRunningStepFooterMainSection(stepRunResult))
		fmt.Printf("+%s+%s+%s+\n", strings.Repeat("-", iconBoxWidth), strings.Repeat("-", titleBoxWidth), strings.Repeat("-", timeBoxWidth))
		if stepRunResult.Error != nil {
			isUpdateAvailable := IsUpdateAvailable(stepRunResult.StepInfo)
			fmt.Println(getRunningStepFooterSubSection(stepRunResult, isUpdateAvailable))
			fmt.Printf("+%s+%s+%s+\n", strings.Repeat("-", iconBoxWidth), strings.Repeat("-", titleBoxWidth), strings.Repeat("-", timeBoxWidth))
		}
	}
	runtime := tmpTime.Sub(time.Time{})

	runtimeStr := TimeToFormattedSeconds(runtime, " sec")
	whitespaceWidth = stepRunSummaryBoxWidthInChars - len(fmt.Sprintf("| Total runtime: %s|", runtimeStr))
	fmt.Printf("| Total runtime: %s%s|\n", runtimeStr, strings.Repeat(" ", whitespaceWidth))
	fmt.Printf("+%s+\n", strings.Repeat("-", stepRunSummaryBoxWidthInChars-2))

	fmt.Println()
}

// PrintStepStatusList ...
func PrintStepStatusList(header string, stepList []models.StepRunResultsModel) {
	if len(stepList) > 0 {
		log.Infof(header)
		for _, stepResult := range stepList {
			stepInfo := stepResult.StepInfo
			if stepResult.Error != nil {
				log.Infof(" * Step: (%s) | error: (%v)", stepInfo.ID, stepResult.Error)
			} else {
				log.Infof(" * Step: (%s)", stepInfo.ID)
			}
		}
	}
}
