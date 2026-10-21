// 一次性迁移工具：给 api_logs 补充播放所需字段。
package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

type column struct {
	name string
	ddl  string
}

func main() {
	dsn := "root:yuchen@tcp(127.0.0.1:3306)/music_admin?charset=utf8mb4&parseTime=true&loc=Local"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("open: %v", err)
	}
	defer db.Close()

	cols := []column{
		{"singer", "ALTER TABLE api_logs ADD COLUMN `singer` VARCHAR(255) DEFAULT '' COMMENT '歌手' AFTER `song_name`"},
		{"song_id", "ALTER TABLE api_logs ADD COLUMN `song_id` VARCHAR(120) DEFAULT '' COMMENT '歌曲ID' AFTER `singer`"},
		{"music_url", "ALTER TABLE api_logs ADD COLUMN `music_url` VARCHAR(1000) DEFAULT '' COMMENT '播放直链' AFTER `album_img`"},
		{"lyric_lrc", "ALTER TABLE api_logs ADD COLUMN `lyric_lrc` MEDIUMTEXT COMMENT 'LRC歌词' AFTER `music_url`"},
	}

	for _, c := range cols {
		var n int
		err := db.QueryRow(`SELECT COUNT(*) FROM information_schema.columns
			WHERE table_schema = 'music_admin' AND table_name = 'api_logs' AND column_name = ?`, c.name).Scan(&n)
		if err != nil {
			log.Fatalf("check %s: %v", c.name, err)
		}
		if n > 0 {
			fmt.Printf("skip %s (exists)\n", c.name)
			continue
		}
		if _, err := db.Exec(c.ddl); err != nil {
			log.Fatalf("alter %s: %v", c.name, err)
		}
		fmt.Printf("ok %s\n", c.name)
	}

	// 历史行这些列为 NULL，先补空串再改为 NOT NULL，避免查询扫描失败
	fixes := []struct{ name, update, ddl string }{
		{"singer", "UPDATE api_logs SET singer = '' WHERE singer IS NULL",
			"ALTER TABLE api_logs MODIFY COLUMN `singer` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '歌手'"},
		{"song_id", "UPDATE api_logs SET song_id = '' WHERE song_id IS NULL",
			"ALTER TABLE api_logs MODIFY COLUMN `song_id` VARCHAR(120) NOT NULL DEFAULT '' COMMENT '歌曲ID'"},
		{"music_url", "UPDATE api_logs SET music_url = '' WHERE music_url IS NULL",
			"ALTER TABLE api_logs MODIFY COLUMN `music_url` VARCHAR(1000) NOT NULL DEFAULT '' COMMENT '播放直链'"},
		{"lyric_lrc", "UPDATE api_logs SET lyric_lrc = '' WHERE lyric_lrc IS NULL",
			"ALTER TABLE api_logs MODIFY COLUMN `lyric_lrc` MEDIUMTEXT NOT NULL COMMENT 'LRC歌词'"},
	}
	for _, f := range fixes {
		if _, err := db.Exec(f.update); err != nil {
			log.Fatalf("backfill %s: %v", f.name, err)
		}
		if _, err := db.Exec(f.ddl); err != nil {
			log.Fatalf("modify %s: %v", f.name, err)
		}
		fmt.Printf("ok %s not null\n", f.name)
	}

	fmt.Println("migrate done")
}
