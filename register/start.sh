#!/bin/bash

USER=root
DIR=/home/nagae-memooff/dns
DAEMON=$DIR/register
PIDFILE=/var/run/my_dns_register.pid
LOGFILE=/var/log/my_dns_register.log

RETVAL=0
prog="my_dns_register"

if [ ! -f $PIDFILE ]; then
  touch $PIDFILE
  chown ${USER}:${USER} $PIDFILE
fi

start () {
  PID=`pidof my_dns_server`
  if [ $? -eq 0 ]; then
    echo "$prog has already been started!"
    exit 0;
  fi
  echo -n $"Starting $prog: "
  start-stop-daemon --start --background --chuid $USER --chdir $DIR --make-pidfile --pidfile $PIDFILE --startas /bin/bash -- -c "exec $DAEMON >> $LOGFILE 2>&1"

  RETVAL=$?
  if [ $RETVAL -eq 0 ]; then
    sleep 1
    chown ${USER}:${USER} $PIDFILE
    echo "ok"
  else
    echo "failed"
  fi
  echo
  return $RETVAL
}

stop () {
  pidof $DAEMON > /dev/null
  if [ $? -ne 0 ]; then
    echo "$prog is not running!"
    exit 0;
  fi

  echo -n $"Stopping $prog: "
  kill `cat $PIDFILE`
  RETVAL=$?
  if [ $RETVAL -eq 0 ] ; then
    echo "ok"
  else
    echo "failed"
  fi
  echo
  return $RETVAL
}

status2 () {
  pid=`pidof $DAEMON`
  if [ $? -eq 0 ]; then
    echo "$prog is running with pid $pid"
    return 0;
  else
    echo "$prog is not running!"
    return 1;
  fi
  return 2
}

restart () {
  stop
  start
}

# See how we were called.
case "$1" in
  start)
    start
    ;;
  stop)
    stop
    ;;
  status)
    status2
    ;;
  restart|reload)
    restart
    ;;
  condrestart)
    [ -f $LOCKFILE ] && restart || :
    ;;
  *)
    echo $"Usage: $0 {start|stop|status|restart|reload|condrestart}"
    exit 1
esac

exit $?
