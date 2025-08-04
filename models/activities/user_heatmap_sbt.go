package activities

import (
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/organization"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/timeutil"
)

// GetUserHeatmapDataSbt метод аналогичен getUserHeatmapDataSbt
// В getUserHeatmapData в хитмапу включены действия, которые совершает только пользователь или все пользователи если это организация.
func GetUserHeatmapDataSbt(user *user_model.User, team *organization.Team, doer *user_model.User, isOnlyPerformed bool, showPrivate bool) ([]*UserHeatmapData, error) {
	hdata := make([]*UserHeatmapData, 0)

	if !ActivityReadable(user, doer) {
		return hdata, nil
	}

	// Group by 15 minute intervals which will allow the client to accurately shift the timestamp to their timezone.
	// The interval is based on the fact that there are timezones such as UTC +5:30 and UTC +12:45.
	groupBy := "created_unix / 900 * 900"
	groupByName := "timestamp" // We need this extra case because mssql doesn't allow grouping by alias
	switch {
	case setting.Database.Type.IsMySQL():
		groupBy = "created_unix DIV 900 * 900"
	case setting.Database.Type.IsMSSQL():
		groupByName = groupBy
	}

	cond, err := activityQueryCondition(GetFeedsOptions{
		RequestedUser:   user,
		RequestedTeam:   team,
		Actor:           doer,
		IncludePrivate:  showPrivate,
		IncludeDeleted:  false,
		OnlyPerformedBy: isOnlyPerformed,
	})
	if err != nil {
		return nil, err
	}

	return hdata, db.GetEngine(db.DefaultContext).
		Select(groupBy+" AS timestamp, count(user_id) as contributions").
		Table("action").
		Where(cond).
		And("created_unix > ?", timeutil.TimeStampNow()-31536000).
		GroupBy(groupByName).
		OrderBy("timestamp").
		Find(&hdata)
}
