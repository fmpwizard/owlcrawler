module.exports = function(grunt) {
  'use strict';
  // Project configuration.
  grunt.initConfig({
    pkg: grunt.file.readJSON('package.json'),
    appFiles : {
      js:  'app/js/component/**/*.js',
      css: 'app/css/*.css'
    },
    dirs: {
      src: 'app/js/',
      target: 'target/grunt',
      dist: '<%= dirs.target %>/dist',
      build: 'build',
      resources: '<%= dirs.target %>/resources'
    },
    delta : {
      gruntfile: {
        files: 'Gruntfile.js',
        tasks: [ 'jshint:gruntfile' ]
      },
      cssmin: {
        files: ['<%= appFiles.css %>'],
        tasks: ['cssmin']
      },
      jssrc: {
        files: ['<%= appFiles.js %>'],
        tasks: [ 'jshint:src', 'requirejs', 'cssmin']
      }
    },
    jshint: {
      options: {
        curly: true,
        eqeqeq: true,
        immed: true,
        latedef: true,
        newcap: true,
        noarg: true,
        sub: true,
        undef: true,
        unused: true,
        boss: true,
        eqnull: true,
        browser: true,
        evil: true,
        globals: {
          'jQuery': false,
          '$': false,
          '_': false,
          'App': true,
          'escape': true
        },
        jshintrc: '.jshintrc'
      },
      src: [
        '<%= appFiles.js %>'
      ],
      gruntfile: {
        options: {
          globals: {
            'module': false,
            'require': false
          }
        },
        files: {
          src: ['Gruntfile.js']
        }
      }
    },
    clean: {
      build: ['<%= dirs.target %>', '<%= dirs.build %>']
    },
    cssmin: {
      combine: {
        files: {
          '<%= dirs.build %>/main.min.css': [
            'app/bower_components/components-bootstrap/css/bootstrap.css',
            'app/bower_components/components-bootstrap/css/bootstrap-theme.css',
            'app/css/main.css'
          ]
        }
      }
    },
    requirejs: {
      compile: {
        options: {
          mainConfigFile: 'app/js/main.js',
          modules: [
            {
              name: 'main',
              out: 'main.min.js'
            }
          ],
          baseUrl: './',
          removeCombined: true,
          dir: '<%= dirs.build %>/',
          appDir: 'app/js',
          optimize: 'uglify',
          uglify: {
            toplevel: true,
            'ascii_only': true,
            beautify: false,
            'max_line_length': 1000,
            //How to pass uglifyjs defined symbols for AST symbol replacement,
            //see "defines" options for ast_mangle in the uglifys docs.
            defines: {
              DEBUG: ['name', 'false']
            },

            //Custom value supported by r.js but done differently
            //in uglifyjs directly:
            //Skip the processor.ast_mangle() part of the uglify call (r.js 2.0.5+)
            'no_mangle': true
          }
        }
      }
    }
  });

  // Load the plugin that provides the 'uglify' task.
  grunt.loadNpmTasks('grunt-contrib-jshint');
  grunt.loadNpmTasks('grunt-contrib-watch');
  grunt.loadNpmTasks('grunt-contrib-clean');
  grunt.loadNpmTasks('grunt-contrib-cssmin');
  grunt.loadNpmTasks('grunt-contrib-requirejs');
  grunt.renameTask('watch', 'delta');
  grunt.registerTask('watch', ['jshint', 'delta']);
  

  // Default task(s).
  grunt.registerTask('default', ['clean', 'jshint', 'requirejs', 'cssmin']);

};