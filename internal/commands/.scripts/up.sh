#!/bin/bash
for ARGUMENT in "$@"
do
  KEY=$(echo $ARGUMENT | cut -f1 -d=)
    VALUE=$(echo $ARGUMENT | cut -f2 -d=)
    case "$KEY" in
            log_rotation_size)              log_rotation_size=${VALUE} ;;
            log_rotation_age_days)    log_rotation_age_days=${VALUE} ;;
            private_key_path)    private_key_path=${VALUE} ;;
            certificate_path)    certificate_path=${VALUE} ;;
            fqdn)    fqdn=${VALUE} ;;
          deploy_DB)    deploy_DB=${VALUE} ;;
          deploy_TLS)    deploy_TLS=${VALUE} ;;
            *)
    esac
done
echo AST_INSTALLATION_PATH=$AST_INSTALLATION_PATH,DATABASE_HOST=$DATABASE_HOST,DATABASE_PORT=$DATABASE_PORT,DATABASE_USER=$DATABASE_USER,DATABASE_PASSWORD=$DATABASE_PASSWORD,DATABASE_INSTANCE=$DATABASE_INSTANCE,ENTRYPOINT_PORT=$ENTRYPOINT_PORT,ENTRYPOINT_TLS_PORT=$ENTRYPOINT_TLS_PORT,TLS_PRIVATE_KEY_PATH=$TLS_PRIVATE_KEY_PATH,TLS_CERTIFICATE_PATH=$TLS_CERTIFICATE_PATH,FQDN=$FQDN,LOG_LEVEL=$LOG_LEVEL,LOG_ROTATION_AGE_DAYS=$LOG_ROTATION_AGE_DAYS,LOG_ROTATION_MAX_SIZE_MB=$LOG_ROTATION_MAX_SIZE_MB,EXTERNAL_ACCESS_IP=$EXTERNAL_ACCESS_IP