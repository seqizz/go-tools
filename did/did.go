package main

// gurkan.in | license: Apache License 2.0 shared on https://github.com/seqizz/go-tools/blob/master/LICENSE

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"database/sql"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mitchellh/go-homedir"
	"github.com/olekukonko/tablewriter"
)

func main() {
	rmFlag := flag.Int("rm", 0, "Remove the specified entry")
	editFlag := flag.String("edit", "", "Edit the specified entry: 3:NewValue")
	showFlag := flag.Bool("show", false, "Show the current done list")
	flag.Parse()

	//can't believe I need this for portability
	findHome, err := homedir.Dir()
	errCheck(err)

	db, err := sql.Open("sqlite3", findHome+"/.did.db")
	errCheck(err)
	defer db.Close()

	sqlStmt := `
	create table if not exists didtable (id integer primary key autoincrement, date string not null, did string not null);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return
	}
	tx, err := db.Begin()
	errCheck(err)

	//if only show is specified, show the contents and exit
	if strings.Join(os.Args[1:], "@") == "show" {
		*showFlag = true
	}

	if *showFlag {
		showAllValues(tx)
		os.Exit(0)
	}

	if *rmFlag == 0 {
		if len(*editFlag) == 0 {
			//not editing nor removing, so let's add
			if len(os.Args) > 1 {
				addValue(strings.Join(os.Args[1:], " "), tx)
			} else {
				infoText()
			}
		} else {
			//editFlag
			mid := strings.Split(strings.Join(os.Args[2:], " "), ":")
			if len(mid) == 2 {
				myid, err := strconv.Atoi(mid[0])
				errCheck(err)
				changeValue(myid, mid[1], tx)
			} else {
				fmt.Println("Please use the format id:text (Example: -edit 3:updated the script)")
				os.Exit(1)
			}
		}
	} else {
		//rmFlag
		rmValue(*rmFlag, tx)
		fmt.Println("Removed", *rmFlag)
	}

}

func errCheck(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}

func addValue(what string, tx *sql.Tx) bool {
	stmt, err := tx.Prepare("insert into didtable(date, did) values(?, ?)")
	errCheck(err)
	const layout = "02 Jan 2006 15:04"
	_, err = stmt.Exec(time.Now().Format(layout), what)
	errCheck(err)
	stmt.Close()
	tx.Commit()
	return true
}

func changeValue(id int, what string, tx *sql.Tx) bool {
	stmt, err := tx.Prepare("update didtable set did = ? where id = ?")
	errCheck(err)
	_, err = stmt.Exec(what, id)
	errCheck(err)
	stmt.Close()
	tx.Commit()
	return true
}

func rmValue(id int, tx *sql.Tx) bool {
	stmt, err := tx.Prepare("delete from didtable where id=?")
	errCheck(err)
	_, err = stmt.Exec(id)
	errCheck(err)
	stmt.Close()
	tx.Commit()
	return true
}

func showAllValues(tx *sql.Tx) bool {
	results, err := tx.Query("select * from didtable")
	errCheck(err)
	defer results.Close()
	tx.Commit()
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"ID", "Date", "What"})
	table.SetCenterSeparator(" ")
	table.SetColumnSeparator(" ")
	table.SetRowLine(false)

	for results.Next() {
		var id int
		var date string
		var what string
		err = results.Scan(&id, &date, &what)
		errCheck(err)
		mid := []string{strconv.Itoa(id), date, what}
		table.Append(mid)
	}
	table.Render()

	return true
}

func infoText() {
	fmt.Println(`
This program is intended to keep a basic version of what happened on the server on multi-admin environments.

Usage:

	did installed python
	did upgraded glibc
	did -show
	did -rm 2
	did -update 1:installed python 2.7
	`)
}
