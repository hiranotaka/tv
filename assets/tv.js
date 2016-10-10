function updatePositions() {
    $('td.main-hour').css('left', $(window).scrollLeft() + 'px');
    $('td.main-program').css('top', $(window).scrollTop() + 'px');
}

function updateMain() {
    $('.main').parent().load(window.location.href + ' .main', function() {
	updatePositions();
    });
}

$(window).scroll(function () {
    updatePositions();
});
updatePositions();

$(window).submit(function(event) {
    var target = $(event.target);
    var action = target.prop('action');
    window.history.pushState(null, null, action);
    $('.main').parent().load(
	url + ' .main', target.serializeArray(), function() {
	    updatePositions();
	});
    $('.event :input').prop('disabled', true);
    event.preventDefault();
});

$(window).click(function(event) {
    var target = $(event.target);
    var href = target.prop('href');
    if (href) {
	window.history.pushState(null, null, href);
	updateMain();
	event.preventDefault();
    }
});

$(window).bind('popstate', function() {
    updateMain();
});
