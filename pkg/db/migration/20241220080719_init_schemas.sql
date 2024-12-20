-- Create "cp_us" table
CREATE TABLE `cp_us` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `timestamp` datetime NOT NULL, `usage` real NOT NULL);
-- Create index "cpu_timestamp" to table: "cp_us"
CREATE INDEX `cpu_timestamp` ON `cp_us` (`timestamp`);
-- Create "cpu_per_hours" table
CREATE TABLE `cpu_per_hours` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `timestamp` datetime NOT NULL, `peak_usage` real NOT NULL, `avg_usage` real NOT NULL);
-- Create index "cpuperhour_timestamp" to table: "cpu_per_hours"
CREATE INDEX `cpuperhour_timestamp` ON `cpu_per_hours` (`timestamp`);
-- Create "disk_ios" table
CREATE TABLE `disk_ios` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `timestamp` datetime NOT NULL, `device` text NOT NULL, `read_bytes` integer NOT NULL, `write_bytes` integer NOT NULL);
-- Create index "diskio_timestamp" to table: "disk_ios"
CREATE INDEX `diskio_timestamp` ON `disk_ios` (`timestamp`);
-- Create "disk_io_per_hours" table
CREATE TABLE `disk_io_per_hours` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `timestamp` datetime NOT NULL, `device` text NOT NULL, `peak_read_bytes` integer NOT NULL, `peak_write_bytes` integer NOT NULL, `avg_read_bytes` integer NOT NULL, `avg_write_bytes` integer NOT NULL);
-- Create index "diskioperhour_timestamp" to table: "disk_io_per_hours"
CREATE INDEX `diskioperhour_timestamp` ON `disk_io_per_hours` (`timestamp`);
-- Create "disk_usages" table
CREATE TABLE `disk_usages` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `timestamp` datetime NOT NULL, `device` text NOT NULL, `mount_point` text NOT NULL, `usage` real NOT NULL, `total` integer NOT NULL, `free` integer NOT NULL, `used` integer NOT NULL);
-- Create index "diskusage_timestamp" to table: "disk_usages"
CREATE INDEX `diskusage_timestamp` ON `disk_usages` (`timestamp`);
-- Create "disk_usage_per_hours" table
CREATE TABLE `disk_usage_per_hours` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `timestamp` datetime NOT NULL, `device` text NOT NULL, `mount_point` text NOT NULL, `peak_usage` real NOT NULL, `avg_usage` real NOT NULL);
-- Create index "diskusageperhour_timestamp" to table: "disk_usage_per_hours"
CREATE INDEX `diskusageperhour_timestamp` ON `disk_usage_per_hours` (`timestamp`);
-- Create "memories" table
CREATE TABLE `memories` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `timestamp` datetime NOT NULL, `usage` real NOT NULL);
-- Create index "memory_timestamp" to table: "memories"
CREATE INDEX `memory_timestamp` ON `memories` (`timestamp`);
-- Create "memory_per_hours" table
CREATE TABLE `memory_per_hours` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `timestamp` datetime NOT NULL, `peak_usage` real NOT NULL, `avg_usage` real NOT NULL);
-- Create index "memoryperhour_timestamp" to table: "memory_per_hours"
CREATE INDEX `memoryperhour_timestamp` ON `memory_per_hours` (`timestamp`);
-- Create "traffics" table
CREATE TABLE `traffics` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `timestamp` datetime NOT NULL, `name` text NOT NULL, `input_pps` real NOT NULL, `input_bps` real NOT NULL, `output_pps` real NOT NULL, `output_bps` real NOT NULL);
-- Create index "traffic_timestamp" to table: "traffics"
CREATE INDEX `traffic_timestamp` ON `traffics` (`timestamp`);
-- Create "traffic_per_hours" table
CREATE TABLE `traffic_per_hours` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `timestamp` datetime NOT NULL, `name` text NOT NULL, `peak_input_pps` real NOT NULL, `peak_input_bps` real NOT NULL, `peak_output_pps` real NOT NULL, `peak_output_bps` real NOT NULL, `avg_input_pps` real NOT NULL, `avg_input_bps` real NOT NULL, `avg_output_pps` real NOT NULL, `avg_output_bps` real NOT NULL);
-- Create index "trafficperhour_timestamp" to table: "traffic_per_hours"
CREATE INDEX `trafficperhour_timestamp` ON `traffic_per_hours` (`timestamp`);
