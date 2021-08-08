FROM mysql/mysql-server:5.7
#https://www.codenong.com/25920029/

ADD ./init/github.sql /var/github.sql

ADD ./init/setup.sh /root/setup.sh

RUN chmod +x /root/setup.sh
EXPOSE 3306

CMD ["/root/setup.sh"]