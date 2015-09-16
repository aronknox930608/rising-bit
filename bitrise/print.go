package bitrise

import (
	"fmt"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/bitrise-io/bitrise/models"
	"github.com/bitrise-io/go-utils/colorstring"
	"github.com/bitrise-io/go-utils/stringutil"
)

const (
	// should not be under ~45
	stepRunSummaryBoxWidthInChars = 65
)

// PrintRunningWorkflow ...
func PrintRunningWorkflow(title string) {
	fmt.Println()
	log.Info(colorstring.Bluef("Running workflow (%s)", title))
	fmt.Println()
}

// PrintRunningStep ...
func PrintRunningStep(stepInfo models.StepInfoModel, idx int) {
	title := stepInfo.ID
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

	sep := strings.Repeat("-", len(content))
	log.Info(sep)
	log.Infof(content)
	log.Info(sep)
	log.Info("|" + strings.Repeat(" ", stepRunSummaryBoxWidthInChars-2) + "|")
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

func stepNoteCell(stepRunResult models.StepRunResultsModel) string {
	iconBoxWidth := len("    ")
	timeBoxWidth := len(" time (s) ")
	titleBoxWidth := stepRunSummaryBoxWidthInChars - 4 - iconBoxWidth - timeBoxWidth - 2

	stepInfo := stepRunResult.StepInfo
	whitespaceWidth := titleBoxWidth - len(fmt.Sprintf("update available %s -> %s", stepInfo.Version, stepInfo.Latest))
	content := colorstring.Yellow(fmt.Sprintf(" Update available: %s -> %s%s", stepInfo.Version, stepInfo.Latest, strings.Repeat(" ", whitespaceWidth)))
	return fmt.Sprintf("|%s|%s|%s|", strings.Repeat("-", iconBoxWidth), content, strings.Repeat("-", timeBoxWidth))
}

func stepResultCell(stepRunResult models.StepRunResultsModel) string {
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

// PrintStepSummary ..
func PrintStepSummary(stepRunResult models.StepRunResultsModel, isLastStepInWorkflow bool) {
	iconBoxWidth := len("    ")
	timeBoxWidth := len(" time (s) ")
	titleBoxWidth := stepRunSummaryBoxWidthInChars - 4 - iconBoxWidth - timeBoxWidth
	sep := fmt.Sprintf("+%s+%s+%s+", strings.Repeat("-", iconBoxWidth), strings.Repeat("-", titleBoxWidth), strings.Repeat("-", timeBoxWidth))

	log.Info("|" + strings.Repeat(" ", stepRunSummaryBoxWidthInChars-2) + "|")

	log.Info(sep)
	log.Infof(stepResultCell(stepRunResult))
	if stepRunResult.Error != nil && stepRunResult.StepInfo.IsUpdateAvailable() {
		log.Info(stepNoteCell(stepRunResult))
	}
	log.Info(sep)

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
	log.Infof("+%s+", strings.Repeat("-", stepRunSummaryBoxWidthInChars-2))
	whitespaceWidth := (stepRunSummaryBoxWidthInChars - 2 - len("bitrise summary")) / 2
	log.Infof("|%sbitrise summary%s|", strings.Repeat(" ", whitespaceWidth), strings.Repeat(" ", whitespaceWidth))
	log.Infof("+%s+%s+%s+", strings.Repeat("-", iconBoxWidth), strings.Repeat("-", titleBoxWidth), strings.Repeat("-", timeBoxWidth))

	whitespaceWidth = stepRunSummaryBoxWidthInChars - len("|    | title") - len("| time (s) |")
	log.Infof("|    | title%s| time (s) |", strings.Repeat(" ", whitespaceWidth))
	log.Infof("+%s+%s+%s+", strings.Repeat("-", iconBoxWidth), strings.Repeat("-", titleBoxWidth), strings.Repeat("-", timeBoxWidth))

	orderedResults := buildRunResults.OrderedResults()
	tmpTime := time.Time{}
	for _, stepRunResult := range orderedResults {
		tmpTime = tmpTime.Add(stepRunResult.RunTime)
		log.Info(stepResultCell(stepRunResult))
		if stepRunResult.Error != nil && stepRunResult.StepInfo.IsUpdateAvailable() {
			log.Info(stepNoteCell(stepRunResult))
		}
		log.Infof("+%s+%s+%s+", strings.Repeat("-", iconBoxWidth), strings.Repeat("-", titleBoxWidth), strings.Repeat("-", timeBoxWidth))
	}
	runtime := tmpTime.Sub(time.Time{})

	runtimeStr := TimeToFormattedSeconds(runtime, " sec")
	whitespaceWidth = stepRunSummaryBoxWidthInChars - len(fmt.Sprintf("| Total runtime: %s|", runtimeStr))
	log.Infof("| Total runtime: %s%s|", runtimeStr, strings.Repeat(" ", whitespaceWidth))
	log.Infof("+%s+", strings.Repeat("-", stepRunSummaryBoxWidthInChars-2))

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
