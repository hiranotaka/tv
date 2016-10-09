function updatePositions() {
    $('td.hour').css('left', $(window).scrollLeft() + 'px');
    $('td.program').css('top', $(window).scrollTop() + 'px');
}

function updateSelectedEvent() {
    $('.event').parent().load(window.location.href + ' .event', null);
}

$(window).scroll(function () {
    updatePositions();
});
updatePositions();

$(window).submit(function(event) {
    var target = $(event.target);
    window.history.pushState(null, null, './?mode=html');
    $('.main').parent().load(
	'./?mode=html .main', target.serializeArray(), function() {
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
	updateSelectedEvent();
	event.preventDefault();
    }
});

$(window).bind('popstate', function() {
    updateSelectedEvent();
});
