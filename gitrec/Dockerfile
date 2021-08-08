# Build frontend
FROM node
COPY frontend frontend
WORKDIR /frontend
RUN npm config set registry https://registry.npm.taobao.org
RUN npm config set disturl https://npm.taobao.org/dist
RUN yarn config set registry https://registry.npm.taobao.org/
RUN yarn config set "strict-ssl" false -g
RUN yarn install && yarn build

# Setup python enviroment
FROM python
COPY backend backend
COPY requirements.txt requirements.txt
COPY gunicorn.conf.py gunicorn.conf.py

# Copy frontend
COPY --from=0 /frontend/dist /frontend/dist

#RUN python -m pip install --upgrade pip

# Install packages  --trusted-host pypi.org --trusted-host pypi.douban.com
RUN pip3 install -r requirements.txt --trusted-host pypi.org --trusted-host files.pythonhosted.org

# Start gitrec
CMD PYTHONPATH=backend gunicorn backend.app:app -c gunicorn.conf.py
