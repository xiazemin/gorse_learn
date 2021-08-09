Create Table: CREATE TABLE `feedback` (
  `feedback_type` varchar(256) NOT NULL,
  `user_id` varchar(256) NOT NULL,
  `item_id` varchar(256) NOT NULL,
  `time_stamp` timestamp NOT NULL,
  `comment` text NOT NULL,
  PRIMARY KEY (`feedback_type`,`user_id`,`item_id`),
  KEY `user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

Create Table: CREATE TABLE `items` (
  `item_id` varchar(256) NOT NULL,
  `time_stamp` timestamp NOT NULL,
  `labels` json NOT NULL,
  `comment` text NOT NULL,
  PRIMARY KEY (`item_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

Create Table: CREATE TABLE `measurements` (
  `name` varchar(256) NOT NULL,
  `time_stamp` timestamp NOT NULL,
  `value` double NOT NULL,
  `comment` text NOT NULL,
  PRIMARY KEY (`name`,`time_stamp`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;


Create Table: CREATE TABLE `users` (
  `user_id` varchar(256) NOT NULL,
  `labels` json NOT NULL,
  `comment` text NOT NULL,
  `subscribe` json NOT NULL,
  PRIMARY KEY (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;


select * from feedback limit 1\G
*************************** 1. row ***************************
feedback_type: like
      user_id: 275761919
      item_id: 720kb:ndm
   time_stamp: 2021-05-26 14:03:35
      comment:
1 row in set (0.00 sec)


select * from items limit 1\G
*************************** 1. row ***************************
   item_id: 00-evan:shattered-pixel-dungeon
time_stamp: 2021-02-08 17:14:41
    labels: ["java"]
   comment: Traditional roguelike game with pixel-art graphics and simple interface
1 row in set (0.00 sec)

select * from measurements limit 1\G
*************************** 1. row ***************************
      name: ActiveUsersMonthly
time_stamp: 2021-07-14 08:00:00
     value: 45
   comment:
1 row in set (0.00 sec)


 select * from users limit 1\G
*************************** 1. row ***************************
  user_id: 0xAX
   labels: null
  comment:
subscribe: null
1 row in set (0.00 sec)


