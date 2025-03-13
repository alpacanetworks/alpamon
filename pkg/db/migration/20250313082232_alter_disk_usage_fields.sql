-- Disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- Create "new_disk_usages" table
CREATE TABLE `new_disk_usages` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `timestamp` datetime NOT NULL, `device` text NOT NULL, `usage` real NOT NULL, `total` integer NOT NULL, `free` integer NOT NULL, `used` integer NOT NULL);
-- Copy rows from old table "disk_usages" to new temporary table "new_disk_usages"
INSERT INTO `new_disk_usages` (`id`, `timestamp`, `device`, `usage`, `total`, `free`, `used`) SELECT `id`, `timestamp`, `device`, `usage`, `total`, `free`, `used` FROM `disk_usages`;
-- Drop "disk_usages" table after copying rows
DROP TABLE `disk_usages`;
-- Rename temporary table "new_disk_usages" to "disk_usages"
ALTER TABLE `new_disk_usages` RENAME TO `disk_usages`;
-- Create index "diskusage_timestamp" to table: "disk_usages"
CREATE INDEX `diskusage_timestamp` ON `disk_usages` (`timestamp`);
-- Create "new_hourly_disk_usages" table
CREATE TABLE `new_hourly_disk_usages` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `timestamp` datetime NOT NULL, `device` text NOT NULL, `peak` real NOT NULL, `avg` real NOT NULL, `total` integer NOT NULL DEFAULT (0), `free` integer NOT NULL DEFAULT (0), `used` integer NOT NULL DEFAULT (0));
-- Copy rows from old table "hourly_disk_usages" to new temporary table "new_hourly_disk_usages"
INSERT INTO `new_hourly_disk_usages` (`id`, `timestamp`, `device`, `peak`, `avg`) SELECT `id`, `timestamp`, `device`, `peak`, `avg` FROM `hourly_disk_usages`;
-- Drop "hourly_disk_usages" table after copying rows
DROP TABLE `hourly_disk_usages`;
-- Rename temporary table "new_hourly_disk_usages" to "hourly_disk_usages"
ALTER TABLE `new_hourly_disk_usages` RENAME TO `hourly_disk_usages`;
-- Create index "hourlydiskusage_timestamp" to table: "hourly_disk_usages"
CREATE INDEX `hourlydiskusage_timestamp` ON `hourly_disk_usages` (`timestamp`);
-- Enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;
