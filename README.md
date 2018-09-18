Link Shortener
==============
### A link shortener built by Wyatt Calandro

Requirements
------------
The link shortener requires two databases:
- MySQL (to store link information)
- Redis (for user sessions)

Environment Variables
---------------------
The link shortener requires certain environment variables to be set in order to function:

| Environment Variable | Description                                                       | Required |
|----------------------|-------------------------------------------------------------------|----------|
| `MYSQL_URI`          | URI for MySQL eg username:password@tcp(127.0.0.1:3306)/dbname     | Yes      |
| `REDIS_HOST`         | Host for Redis server eg 127.0.0.1:6379                           | Yes      |
| `REDIS_PASSWORD`     | Password for Redis server                                         | Yes      |
| `WEBSITE_URL`        | Domain for website where you create URLs eg "links.wcalandro.com" | Yes      |
| `SHORT_URL`          | Domain for short URLs eg "wcal.xyz"                               | Yes      |
| `SESSION_SECRET`     | Secret used to encrypt sessions                                   | Yes      |

Running the application
-----------------------
This website was built to run on [Heroku](https://heroku.com) or [Dokku](https://github.com/dokku/dokku) and includes a `Dockerfile` that will be detected and built automatically