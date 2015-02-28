package parse

import (
	"testing"
)

var doc1 = `<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta http-equiv="X-UA-Compatible" content="IE=edge">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <meta name="description" content="Psychologist in Hendersonville North Carolina, near Etowah North Carolina, Asheville North Carolina and Brevard North Carolina Psychologist in Hendersonville, North Carolina">
    <meta name="author" content="Dr. Hayley Bauman">

    <title>Psychologist - Asheville - Hendersonville - North Carolina - Hayley J. Bauman, Psy.D - Therapy - Etowah - Brevard</title>

    <link rel="stylesheet" href="/bootstrap/css/bootstrap.min.css">
    <link rel="stylesheet" href="/bootstrap/css/bootstrap-theme.min.css">
    <link rel="stylesheet" href="/site.css">

    <!-- HTML5 shim and Respond.js for IE8 support of HTML5 elements and media queries -->
    <!--[if lt IE 9]>
      <script src="https://oss.maxcdn.com/html5shiv/3.7.2/html5shiv.min.js"></script>
      <script src="https://oss.maxcdn.com/respond/1.4.2/respond.min.js"></script>
    <![endif]-->
  </head>

  <body role="document">

    <!-- Fixed navbar -->
    <nav class="navbar navbar-inverse navbar-fixed-top" role="navigation">
      <div class="container">
        <div class="navbar-header">
          <button type="button" class="navbar-toggle collapsed" data-toggle="collapse" data-target="#navbar" aria-expanded="false" aria-controls="navbar">
            <span class="sr-only">Toggle navigation</span>
            <span class="icon-bar"></span>
            <span class="icon-bar"></span>
            <span class="icon-bar"></span>
          </button>
        </div>
        <div id="navbar" class="navbar-collapse collapse">
          <ul class="nav navbar-nav">
            <li class="active"><a href="#">Home</a></li>
            <li><a href="/educationandtraining.html">Education and Training</a></li>
            <li><a href="/frequentlyaskedquestions.html">Frequently Asked Questions</a></li>
            <li><a href="/psychologyresources.html">Helpful Resources</a></li>
            <li><a href="/gettingstarted.html">Getting Started</a></li>
            <li><a href="/serendipity_and_the_search_for_true_self.html">Serendpity and the Search for True Self</a></li>
            <li><a href="/contact.html">Contact</a></li>
          </ul>
        </div><!--/.nav-collapse -->
      </div>
    </nav>

    <div class="container theme-showcase text-block" role="main">
      <div class="row">
        <div class="col-sm-12 jumbotron">
          <div class="jumbotron">
            <h1>Hayley J. Bauman, Psy.D</h1>
            <h2>Licensed Psychologist</h2>
            <blockquote class="blockquote-reverse">
              <p>"Our deepest fear is not that we are inadequate. Our deepest fear is that we are powerful beyond measure."</p>
              <footer>Marianne Williamsons</footer>
            </blockquote>
          </div>
        </div>
      </div>
      <div class="row">
        <div class="col-sm-12">
          <p>Dr. Hayley Bauman is a licensed clinical psychologist in Etowah, North Carolina, central to the Brevard, Hendersonville, and Asheville areas. Dr. Bauman has more than ten years of experience conducting individual, group, couples, and family therapy with clients from a variety of multi-cultural environments and settings. Her extensive background ranges from working in outpatient private practice, to her role as psychologist, team leader, and supervisor for The Renfrew Center, a nationally known eating disorder treatment facility. She is well versed in the areas of eating disorders, body image, self-esteem, depression, anxiety, relationship stress, and transitional issues.</p>

          <p>Dr. Bauman currently practices out of her home, where she uses a psycho-spiritually oriented approach to assist others in resolving uncomfortable issues, so that they may enjoy a greater sense of peace and balance in their lives. Her book, 
          <em>Serendipity and the Search for True Self,</em> is a whimsical and warm-hearted journey down the winding road of self-discovery, and may be purchased in many wonderful local stores, or online at <a href="http://store.vervante.com/c/v/V4081308950.html">Vervante.com</a>, <a href="http://search.barnesandnoble.com/Hayley-J-Bauman/e/9781607027676">BarnesandNoble.com</a>, or <a href="http://www.amazon.com/Serendipity-Search-Psy-D-Hayley-Bauman/dp/1607027674">Amazon.com</a>.</p>
          
        </div>
      </div>


      

    </div> <!-- /container -->


    <!-- Bootstrap core JavaScript
    ================================================== -->
    <!-- Placed at the end of the document so the pages load faster -->
    <script src="https://ajax.googleapis.com/ajax/libs/jquery/1.11.1/jquery.min.js"></script>
    <script src="/bootstrap/js/bootstrap.min.js"></script>
    <script>
  (function(i,s,o,g,r,a,m){i['GoogleAnalyticsObject']=r;i[r]=i[r]||function(){
  (i[r].q=i[r].q||[]).push(arguments)},i[r].l=1*new Date();a=s.createElement(o),
  m=s.getElementsByTagName(o)[0];a.async=1;a.src=g;m.parentNode.insertBefore(a,m)
  })(window,document,'script','//www.google-analytics.com/analytics.js','ga');

  ga('create', 'UA-368436-5', 'auto');
  ga('send', 'pageview');

</script>
  </body>
</html>`

func mockedFetchChecker(url string) bool {
	return true
}

func TestExtractText(t *testing.T) {
	extracted := ExtractText(doc1)
	if extracted.Title != "Psychologist - Asheville - Hendersonville - North Carolina - Hayley J. Bauman, Psy.D - Therapy - Etowah - Brevard" {
		t.Errorf("ExtractText didn't give us expected result. It gave: %s\n", extracted.Title)
	}
	if len(extracted.H1) != 1 {
		t.Errorf("ExtractText didn't give us expected result: \n%+v\n\nIt gave: %d H1 elements\n", extracted, len(extracted.H1))
	}
	if len(extracted.H2) != 1 {
		t.Errorf("ExtractText didn't give us expected result: \n%+v\n\nIt gave: %d H2 elements\n", extracted, len(extracted.H2))
	}
}

func TestExtractLinks(t *testing.T) {
	extracted := ExtractLinks(doc1, "http://drhayleybauman.com", mockedFetchChecker)
	if len(extracted.URL) != 6 {
		t.Errorf("ExtractLinks didn't give us expected result. It gave: %d urls\n", len(extracted.URL))
	}
}
