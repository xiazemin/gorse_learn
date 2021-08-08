1,standard_init_linux.go:219: exec user process caused: exec format error
https://blog.csdn.net/HermitSun/article/details/109145856

2，进入运行中的容器
https://www.jianshu.com/p/e2dfb2e7ff3c

3，拷贝文件到容器
https://www.cnblogs.com/areyouready/p/8973495.html

4,ERROR 1273 (HY000) at line 25: Unknown collation: 'utf8mb4_0900_ai_ci'
https://blog.csdn.net/qq_41433183/article/details/94772632

5.vue
https://cn.vuejs.org/v2/guide/
https://www.runoob.com/vue2/vue-install.html
npm install -g vue
npm install --global vue-cli

在 App.vue 文件中，已经用一句 @import "./style/style"; 将样式给写到指定的地方去了。所有样式，都会在 src/style/ 文件夹中对应的位置去写。这样做的好处是，不需要重复的引入各种 scss 基础文件，并且做到了项目的样式代码的可管控。
*.vue 文件代码解析
https://jingyan.baidu.com/article/414eccf6a68a956b421f0a60.html
https://zhuanlan.zhihu.com/p/71748090

6.yarn.lock 有什么用
为了跨机器安装得到一致的结果，Yarn 需要比你配置在 package.json 中的依赖列表更多的信息。 Yarn 需要准确存储每个安装的依赖是哪个版本。
为了做到这样，Yarn 使用一个你项目根目录里的 yarn.lock 文件。这可以媲美其他像 Bundler 或 Cargo 这样的包管理器的 lockfiles。它类似于 npm 的 npm-shrinkwrap.json，然而他并不是有损的并且它能创建可重现的结果。
https://www.cnblogs.com/yangzhou33/p/11494819.html
https://yarn.bootcss.com/docs/yarn-lock/

7.babel.config.js
https://blog.csdn.net/weixin_42472040/article/details/112173176
Babel是一个JS编译器，主要作用是将ECMAScript 2015+ 版本的代码，转换为向后兼容的JS语法，以便能够运行在当前和旧版本的浏览器或其它环境中。

8,sh: vue-cli-service: command not found
npm ERR! code ELIFECYCLE
https://www.jianshu.com/p/167200039902
https://blog.csdn.net/wjx666666/article/details/102315060

npm install vue-cli
npm run build

9,Proxy error: Could not proxy request /api/repo from localhost:8080 to http://127.0.0.1:5000.
See https://nodejs.org/api/errors.html#errors_common_system_errors for more information (ECONNREFUSED).

    "serve": "vue-cli-service serve",
    "build": "vue-cli-service build",
    "lint": "vue-cli-service lint",

    "start": "node index.js",
    "server": "nodemon index.js --ignore client"

https://blog.csdn.net/Reagan_/article/details/97498160

https://cli.vuejs.org/zh/config/#babel


10.gunicorn
Gunicorn 绿色独角兽'是一个Python WSGI UNIX的HTTP服务器。这是一个pre-fork worker的模型，从Ruby的独角兽（Unicorn ）项目移植。该Gunicorn服务器大致与各种Web框架兼容，只需非常简单的执行，轻量级的资源消耗，以及相当迅速。
https://www.oschina.net/p/gunicorn?hmsr=aladdin1e1

https://www.jianshu.com/p/69e75fc3e08e

pip3 install gunicorn
gunicorn backend.app:app -c gunicorn.conf.py

11.Error: class uri 'gevent' invalid or not found:
[Traceback (most recent call last):
  File "/usr/local/lib/python3.9/site-packages/gunicorn/workers/ggevent.py", line 13, in <module>

pip3 install gevent
pip3 install -r requirements.txt
PYTHONPATH=backend gunicorn backend.app:app -c gunicorn.conf.py