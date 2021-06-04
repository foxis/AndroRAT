FROM thyrlian/android-sdk as builder
COPY requirements.txt .
RUN apt update && apt install -y pip && pip install -r requirements.txt
WORKDIR code
ENTRYPOINT ["python3", "androRAT.py"]