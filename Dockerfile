FROM openeuler/openeuler:23.03 as BUILDER
RUN dnf update -y && \
    dnf install -y golang && \
    go env -w GOPROXY=https://goproxy.cn,direct

# build binary
COPY . /go/src/mindspore/xihe-script

RUN cd /go/src/mindspore/xihe-script && CGO_ENABLED=1 CGO_CFLAGS="-fstack-protector-all -D_FORTIFY_SOURCE=2 -O2" go build -buildmode=pie --ldflags "-s -linkmode 'external' -extldflags '-Wl,-z,relro,-z,now,-z,noexecstack'" -o xihe-script

FROM openeuler/openeuler:22.03

RUN yum update -y && yum install -y shadow python3 python3-pip

RUN mkdir -p /opt/app/xihe-script/py/data

COPY ./py /opt/app/xihe-script/py

RUN chmod 755 -R /opt/app/xihe-script/py/*.py

ENV EVALUATE /opt/app/xihe-script/py/evaluate.py
ENV CALCULATE /opt/app/xihe-script/py/calculate_fid.py
ENV UPLOAD /opt/app/xihe-script/py/data/

RUN pip config set global.index-url https://pypi.tuna.tsinghua.edu.cn/simple

RUN pip install -r /opt/app/xihe-script/py/requirements.txt


RUN useradd mindspore -u 5000 && chown -R mindspore /opt/app/xihe-script
USER mindspore
WORKDIR /opt/app/xihe-script/

COPY --chown=mindspore:mindspore --from=BUILDER /go/src/mindspore/xihe-script/xihe-script /opt/app/xihe-script/
RUN chmod 550 /opt/app/xihe-script/

ENTRYPOINT ["/opt/app/xihe-script/xihe-script"]
