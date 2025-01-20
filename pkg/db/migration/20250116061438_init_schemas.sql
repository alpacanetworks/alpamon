-- Create "cp_us" table
CREATE TABLE `cp_us` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `timestamp` datetime NOT NULL, `usage` real NOT NULL);
-- Create index "cpu_timestamp" to table: "cp_us"
CREATE INDEX `cpu_timestamp` ON `cp_us` (`timestamp`);
-- Create "disk_ios" table
CREATE TABLE `disk_ios` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `timestamp` datetime NOT NULL, `device` text NOT NULL, `read_bps` real NOT NULL, `write_bps` real NOT NULL);
-- Create index "diskio_timestamp" to table: "disk_ios"
CREATE INDEX `diskio_timestamp` ON `disk_ios` (`timestamp`);
-- Create "disk_usages" table
CREATE TABLE `disk_usages` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `timestamp` datetime NOT NULL, `device` text NOT NULL, `mount_point` text NOT NULL, `usage` real NOT NULL, `total` integer NOT NULL, `free` integer NOT NULL, `used` integer NOT NULL);
-- Create index "diskusage_timestamp" to table: "disk_usages"
CREATE INDEX `diskusage_timestamp` ON `disk_usages` (`timestamp`);
-- Create "hourly_cpu_usages" table
CREATE TABLE `hourly_cpu_usages` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `timestamp` datetime NOT NULL, `peak` real NOT NULL, `avg` real NOT NULL);
-- Create index "hourlycpuusage_timestamp" to table: "hourly_cpu_usages"
CREATE INDEX `hourlycpuusage_timestamp` ON `hourly_cpu_usages` (`timestamp`);
-- Create "hourly_disk_ios" table
CREATE TABLE `hourly_disk_ios` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `timestamp` datetime NOT NULL, `device` text NOT NULL, `peak_read_bps` real NOT NULL, `peak_write_bps` real NOT NULL, `avg_read_bps` real NOT NULL, `avg_write_bps` real NOT NULL);
-- Create index "hourlydiskio_timestamp" to table: "hourly_disk_ios"
CREATE INDEX `hourlydiskio_timestamp` ON `hourly_disk_ios` (`timestamp`);
-- Create "hourly_disk_usages" table
CREATE TABLE `hourly_disk_usages` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `timestamp` datetime NOT NULL, `device` text NOT NULL, `peak` real NOT NULL, `avg` real NOT NULL);
-- Create index "hourlydiskusage_timestamp" to table: "hourly_disk_usages"
CREATE INDEX `hourlydiskusage_timestamp` ON `hourly_disk_usages` (`timestamp`);
-- Create "hourly_memory_usages" table
CREATE TABLE `hourly_memory_usages` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `timestamp` datetime NOT NULL, `peak` real NOT NULL, `avg` real NOT NULL);
-- Create index "hourlymemoryusage_timestamp" to table: "hourly_memory_usages"
CREATE INDEX `hourlymemoryusage_timestamp` ON `hourly_memory_usages` (`timestamp`);
-- Create "hourly_traffics" table
CREATE TABLE `hourly_traffics` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `timestamp` datetime NOT NULL, `name` text NOT NULL, `peak_input_pps` real NOT NULL, `peak_input_bps` real NOT NULL, `peak_output_pps` real NOT NULL, `peak_output_bps` real NOT NULL, `avg_input_pps` real NOT NULL, `avg_input_bps` real NOT NULL, `avg_output_pps` real NOT NULL, `avg_output_bps` real NOT NULL);
-- Create index "hourlytraffic_timestamp" to table: "hourly_traffics"
CREATE INDEX `hourlytraffic_timestamp` ON `hourly_traffics` (`timestamp`);
-- Create "memories" table
CREATE TABLE `memories` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `timestamp` datetime NOT NULL, `usage` real NOT NULL);
-- Create index "memory_timestamp" to table: "memories"
CREATE INDEX `memory_timestamp` ON `memories` (`timestamp`);
-- Create "traffics" table
CREATE TABLE `traffics` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `timestamp` datetime NOT NULL, `name` text NOT NULL, `input_pps` real NOT NULL, `input_bps` real NOT NULL, `output_pps` real NOT NULL, `output_bps` real NOT NULL);
-- Create index "traffic_timestamp" to table: "traffics"
CREATE INDEX `traffic_timestamp` ON `traffics` (`timestamp`);
