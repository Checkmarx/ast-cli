#!/bin/bash
for ARGUMENT in "$@"
do
  KEY=$(echo $ARGUMENT | cut -f1 -d=)
    VALUE=$(echo $ARGUMENT | cut -f2 -d=)
    case "$KEY" in
            log_rotation_size)              log_rotation_size=${VALUE} ;;
            log_rotation_age_days)    log_rotation_age_days=${VALUE} ;;
            tls_private_key_file)    tls_private_key_file=${VALUE} ;;
            tls_certificate_file)    tls_certificate_file=${VALUE} ;;
            *)
    esac
done
echo $log_rotation_size
echo $log_rotation_age_days
echo $tls_private_key_file
echo $tls_certificate_file