package loop

import (
	"github.com/robfig/cron/v3"
)

/*
분 시 일 월 요일

c.AddFunc("30 * * * *", func() { fmt.Println("Every hour on the half hour") })
c.AddFunc("30 3-6,20-23 * * *", func() { fmt.Println(".. in the range 3-6am, 8-11pm") })
c.AddFunc("CRON_TZ=Asia/Tokyo 30 04 * * *", func() { fmt.Println("Runs at 04:30 Tokyo time every day") })
c.AddFunc("@hourly",      func() { fmt.Println("Every hour, starting an hour from now") })
c.AddFunc("@every 1h30m", func() { fmt.Println("Every hour thirty, starting an hour thirty from now") })

Field name   | Mandatory? | Allowed values  | Allowed special characters
----------   | ---------- | --------------  | --------------------------
Minutes      | Yes        | 0-59            | * / , -
Hours        | Yes        | 0-23            | * / , -
Day of month | Yes        | 1-31            | * / , - ?
Month        | Yes        | 1-12 or JAN-DEC | * / , -
Day of week  | Yes        | 0-6 or SUN-SAT  | * / , - ?

Entry                  | Description                                | Equivalent To
-----                  | -----------                                | -------------
@yearly (or @annually) | Run once a year, midnight, Jan. 1st        | 0 0 1 1 *
@monthly               | Run once a month, midnight, first of month | 0 0 1 * *
@weekly                | Run once a week, midnight between Sat/Sun  | 0 0 * * 0
@daily (or @midnight)  | Run once a day, midnight                   | 0 0 * * *
@hourly                | Run once an hour, beginning of hour        | 0 * * * *
*/

func SetupCronJobs() {
	c := cron.New()

	// 매 시간마다 10분
	c.AddFunc("10 * * * *", func() {
		hourlyTask()
	})

	// 2시간마다 (짝수 시간: 0, 2, 4, 6, 8, 10, 12, 14, 16, 18, 20, 22시) 20분
	c.AddFunc("20 0/2 * * *", func() {
		hour2Task()
	})

	// 4시간마다 (0, 4, 8, 12, 16, 20시) 30분
	c.AddFunc("30 0/4 * * *", func() {
		hour4Task()
	})

	// 6시간마다 (0, 6, 12, 18시) 40분
	c.AddFunc("40 0/6 * * *", func() {
		hour6Task()
	})

	// 8시간마다 (0, 8, 16시) 50분
	c.AddFunc("50 0/8 * * *", func() {
		hour8Task()
	})

	// 12시간마다 (0, 12시) 55분
	c.AddFunc("55 0/12 * * *", func() {
		hour12Task()
	})

	// 매일 오전 4시 10분
	c.AddFunc("15 4 * * *", func() {
		dailyTask()
	})

	// 매주 일요일 4시 25분
	c.AddFunc("25 4 * * 0", func() {
		weeklyTask()
	})

	// 매월 1일 4시 35분
	c.AddFunc("35 4 1 * *", func() {
		monthlyTask()
	})

	c.Start()
}
