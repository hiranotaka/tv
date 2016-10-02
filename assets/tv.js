function updatePositions() {
    $('td.hour').css('left', $(window).scrollLeft() + 'px');
    $('td.program').css('top', $(window).scrollTop() + 'px');
}

$(window).scroll(function () {
    updatePositions();
});
updatePositions();

$(window).submit(function(event) {
    var target = $(event.target);
    $('.main').parent().load(
	'./?mode=html .main', target.serializeArray(), function() {
	    updatePositions();
	});
    $('.main :input').prop('disabled', true);
    event.preventDefault();
});
