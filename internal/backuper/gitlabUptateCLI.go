package backuper

import (
	"strconv"
)

func (b BackupCreator) buildGitalyCreateBackupCLICommand(outputBackupDir, backupConfigPath string, numOfThreads int, incremental bool) []string {
	if incremental {
		return []string{"create", "-path", outputBackupDir, "-incremental", "-parallel", strconv.Itoa(numOfThreads), "<", backupConfigPath}
	}
	return []string{"create", "-path", outputBackupDir, "-parallel", strconv.Itoa(numOfThreads), "<", backupConfigPath}
}

func (b BackupRestorer) buildGitalyRestoreBackupCLIACommand(inputBackupDir, backupConfigPath string, numOfThreads int) []string {
	return []string{"restore", "-path", inputBackupDir, "-parallel", strconv.Itoa(numOfThreads), "<", backupConfigPath}
}
