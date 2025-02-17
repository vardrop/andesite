package main

import (
	"database/sql"
	"strconv"

	. "github.com/nektro/go-util/alias"
	. "github.com/nektro/go-util/util"
)

func scanUser(rows *sql.Rows) UserRow {
	var v UserRow
	rows.Scan(&v.ID, &v.Snowflake, &v.Admin, &v.Name)
	return v
}

func scanAccessRow(rows *sql.Rows) UserAccessRow {
	var v UserAccessRow
	rows.Scan(&v.ID, &v.User, &v.Path)
	return v
}

//
//

func queryAccess(user UserRow) []string {
	result := []string{}
	rows := database.Query(false, F("select * from access where user = '%d'", user.ID))
	for rows.Next() {
		result = append(result, scanAccessRow(rows).Path)
	}
	rows.Close()
	return result
}

func queryUserBySnowflake(snowflake string) (UserRow, bool) {
	var ur UserRow
	rows := database.Query(false, F("select * from users where snowflake = '%s'", oauth2Provider.DbP+snowflake))
	if !rows.Next() {
		return ur, false
	}
	ur = scanUser(rows)
	rows.Close()
	ur.Snowflake = ur.Snowflake[len(oauth2Provider.DbP):]
	return ur, true
}

func queryUserByID(id int) (UserRow, bool) {
	var ur UserRow
	rows := database.Query(false, F("select * from users where id = '%d'", id))
	if !rows.Next() {
		return ur, false
	}
	rows.Scan(&ur.ID, &ur.Snowflake, &ur.Admin, &ur.Name)
	rows.Close()
	ur.Snowflake = ur.Snowflake[len(oauth2Provider.DbP):]
	return ur, true
}

func queryAllAccess() []map[string]string {
	var result []map[string]string
	rows := database.Query(false, "select * from access")
	accs := []UserAccessRow{}
	for rows.Next() {
		accs = append(accs, scanAccessRow(rows))
	}
	rows.Close()
	ids := map[int][]string{}
	for _, uar := range accs {
		if _, ok := ids[uar.User]; !ok {
			uu, _ := queryUserByID(uar.User)
			ids[uar.User] = []string{uu.Snowflake, uu.Name}
		}
		result = append(result, map[string]string{
			"id":        strconv.Itoa(uar.ID),
			"user":      strconv.Itoa(uar.User),
			"snowflake": ids[uar.User][0],
			"name":      ids[uar.User][1],
			"path":      uar.Path,
		})
	}
	return result
}

func queryDoAddUser(id int, snowflake string, admin bool, name string) {
	database.QueryPrepared(true, F("insert into users values ('%d', '%s', '%s', ?)", id, oauth2Provider.DbP+snowflake, boolToString(admin)), name)
}

func queryDoUpdate(table string, col string, value string, where string, search string) {
	database.QueryPrepared(true, F("update %s set %s = ? where %s = ?", table, col, where), value, search)
}

func queryAssertUserName(snowflake string, name string) {
	_, ok := queryUserBySnowflake(snowflake)
	if ok {
		queryDoUpdate("users", "name", name, "snowflake", oauth2Provider.DbP+snowflake)
	} else {
		uid := database.QueryNextID("users")
		queryDoAddUser(uid, snowflake, false, name)

		if uid == 0 {
			// always admin first user
			database.QueryDoUpdate("users", "admin", "1", "id", "0")
			aid := database.QueryNextID("access")
			database.Query(true, F("insert into access values ('%d', '%d', '/')", aid, uid))
			Log(F("Set user '%s's status to admin", snowflake))
		}
	}
}

func queryAllShares() []map[string]string {
	var result []map[string]string
	rows := database.Query(false, "select * from shares")
	for rows.Next() {
		var sr ShareRow
		rows.Scan(&sr.ID, &sr.Hash, &sr.Path)
		result = append(result, map[string]string{
			"id":   strconv.Itoa(sr.ID),
			"hash": sr.Hash,
			"path": sr.Path,
		})
	}
	rows.Close()
	return result
}

func queryAllSharesByCode(code string) []ShareRow {
	shrs := []ShareRow{}
	rows := database.QueryPrepared(false, "select * from shares where hash = ?", code)
	for rows.Next() {
		var sr ShareRow
		rows.Scan(&sr.ID, &sr.Hash, &sr.Path)
		shrs = append(shrs, sr)
	}
	rows.Close()
	return shrs
}

func queryAccessByShare(code string) []string {
	result := []string{}
	for _, item := range queryAllSharesByCode(code) {
		result = append(result, item.Path)
	}
	return result
}
