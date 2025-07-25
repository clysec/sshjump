FROM scratch

COPY sshjump /sshjump

ENV ALLOWED_HOSTS=""
ENV SSHD_PORT=2222
ENTRYPOINT [ "/sshjump" ]