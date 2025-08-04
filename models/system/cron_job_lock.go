package system

import (
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit"
	"fmt"
	"os"
	"time"
	"xorm.io/builder"
)

// CronJobLock описывает cron_job_lock таблицу
type CronJobLock struct {
	ID                   int64  `xorm:"pk autoincr"`
	Name                 string `xorm:"unique"`
	Status               string
	Next                 time.Time
	Prev                 time.Time
	Lock                 bool
	HostName             string
	HostIp               string
	LastFailedRun        time.Time
	LastFailedRunMessage string
	ProcessUUID          string
}

// ErrLockCronJob ошибка блокировки крон работы
type ErrLockCronJob struct {
	Name string
}

// IsErrLockCronJob проверяет что ошибка ErrLockCronJob
func IsErrLockCronJob(err error) bool {
	_, ok := err.(ErrLockCronJob)
	return ok
}

// Error выводит сообщение об ошибке ErrLockCronJob
func (err ErrLockCronJob) Error() string {
	return fmt.Sprintf("Cron task with name '%s' is already locked", err.Name)
}

var hostIp = audit.GetLocalIP()
var hostName = ""
var cronJobIdMap = make(map[string]int64, 0)

func init() {
	db.RegisterModel(new(CronJobLock))
	currentHostName, err := os.Hostname()
	if err != nil {
		currentHostName = audit.EmptyRequiredField
	}
	hostName = currentHostName
}

// RegisterCronJob регистрирует крон работу в таблице cron_job_lock
func RegisterCronJob(name string, next time.Time) error {
	if _, ok := cronJobIdMap[name]; ok {
		return nil
	}
	cronJob := &CronJobLock{
		Name:   name,
		Lock:   false,
		Status: "registered",
		Next:   next,
	}
	_, err := db.GetEngine(db.DefaultContext).Insert(cronJob)
	cronJobIdMap[cronJob.Name] = cronJob.ID
	return err
}

// LockCronJob накладывает блокировку на запуск крон работы через таблицу cron_job_lock
func LockCronJob(name string, prev time.Time, processUUID string) error {
	cronJob, err := GetCronJobLock(name)
	if err != nil {
		return err
	}
	if cronJob.Lock {
		failedRunTime := time.Now()
		if failedRunTime.Sub(cronJob.Prev).Seconds() >= 30 {
			_, err := db.GetEngine(db.DefaultContext).
				Where(builder.Eq{"name": name}).
				Update(&CronJobLock{LastFailedRun: time.Now(), LastFailedRunMessage: "Задача не была запущена"})
			if err != nil {
				return err
			}
		}
		return ErrLockCronJob{Name: name}
	}

	cronJob.Lock = true
	cronJob.Prev = prev
	cronJob.Status = "process"
	hostName, err := os.Hostname()
	if err != nil {
		hostName = audit.EmptyRequiredField
	}
	cronJob.HostName = hostName
	cronJob.HostIp = audit.GetLocalIP()
	cronJob.ProcessUUID = processUUID

	count, err := db.GetEngine(db.DefaultContext).
		Where(builder.Eq{"name": name, "lock": false}).
		Cols("status", "prev", "lock", "host_name", "host_ip", "process_uuid").
		Update(cronJob)
	if err != nil {
		return err
	}
	if count == 0 {
		lockedCronJob, err := GetCronJobLock(name)
		if err != nil {
			return err
		}

		failedRunTime := time.Now()
		if failedRunTime.Sub(lockedCronJob.Prev).Seconds() >= 30 {
			_, err := db.GetEngine(db.DefaultContext).
				Where(builder.Eq{"name": name}).
				Update(&CronJobLock{LastFailedRun: time.Now(), LastFailedRunMessage: "Задача не была запущена"})
			if err != nil {
				return err
			}
		}
		return ErrLockCronJob{Name: name}
	}

	log.Debug("Lock task: %s", name)
	return nil
}

// UnlockCronJob снимает блокировку на запуск крон работы через таблицу cron_job_lock
func UnlockCronJob(name string, next time.Time, status string, processUUID string) (int64, error) {
	cronJob := &CronJobLock{
		Status: status,
		Next:   next,
		Lock:   false,
	}
	count, err := db.GetEngine(db.DefaultContext).
		Where(builder.Eq{"name": name, "host_name": hostName, "host_ip": hostIp, "process_uuid": processUUID}).
		Cols("status", "next", "lock", "host_name", "host_ip", "last_failed_run", "last_failed_run_message", "process_uuid").
		Update(cronJob)
	return count, err
}

// ForceUnlockCronJob принудительно снимает блокировку на запуск крон работы через таблицу cron_job_lock
func ForceUnlockCronJob(name string) error {
	cronJob := &CronJobLock{
		Status: "finished",
		Lock:   false,
	}
	_, err := db.GetEngine(db.DefaultContext).
		Where(builder.Eq{"name": name}).
		Cols("status", "lock", "host_name", "host_ip", "last_failed_run", "last_failed_run_message", "process_uuid").
		Update(cronJob)
	return err
}

// LoadCronJobs загружает зарегистрированные блокировки крон работ
func LoadCronJobs() ([]*CronJobLock, error) {
	var cronJobs []*CronJobLock
	return cronJobs, db.GetEngine(db.DefaultContext).Find(&cronJobs)
}

// GetCronJobLock возвращает информацию о блокировке крон работы по её имени
func GetCronJobLock(name string) (*CronJobLock, error) {
	cronJobLock := &CronJobLock{Name: name}
	has, err := db.GetEngine(db.DefaultContext).Get(cronJobLock)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, nil
	}
	return cronJobLock, nil
}

// InitCronJobList инициализирует список блокировок крон работ
func InitCronJobList() error {
	jobs, err := LoadCronJobs()
	if err != nil {
		return err
	}
	for _, job := range jobs {
		cronJobIdMap[job.Name] = job.ID
	}

	return nil
}
