package parcels

// https://habr.com/ru/post/114947/
// transcription https://www.study.ru/article/fonetika-angliyskogo/transkripciya-i-pravila-chteniya
// https://iloveenglish.ru/stories/view/vse-o-transkriptsii-v-anglijskom-yazike
// https://www.translate.ru/Gramm/Rules/
// https://sloovo.com/ru/biblioteka.php?type=obuchenie&language=EN&category=spravochnik&url=angliyskaya-transkripciya-i-pravila-chteniya
// https://ru.wikipedia.org/wiki/%D0%90%D0%BD%D0%B3%D0%BB%D0%BE-%D1%80%D1%83%D1%81%D1%81%D0%BA%D0%B0%D1%8F_%D0%BF%D1%80%D0%B0%D0%BA%D1%82%D0%B8%D1%87%D0%B5%D1%81%D0%BA%D0%B0%D1%8F_%D1%82%D1%80%D0%B0%D0%BD%D1%81%D0%BA%D1%80%D0%B8%D0%BF%D1%86%D0%B8%D1%8F

// hyphenation https://github.com/mnater/hyphenator

var (
	// voiced: muffled
	mRu1 = map[rune]rune{
		'б': 'п',
		'з': 'с',
		'д': 'т',
		'в': 'ф',
		'г': 'к',
	}

	// Consonants except Л, М, Н, Р
	mRu2 = map[rune]bool{
		'б': true,
		'в': true,
		'г': true,
		'д': true,
		'ж': true,
		'з': true,
		'й': true,
		'к': true,
		'п': true,
		'с': true,
		'т': true,
		'ф': true,
		'х': true,
		'ц': true,
		'ч': true,
		'ш': true,
		'щ': true,
	}

	// All consonants
	mRu3 = map[rune]bool{
		'б': true,
		'в': true,
		'г': true,
		'д': true,
		'ж': true,
		'з': true,
		'й': true,
		'к': true,
		'л': true,
		'м': true,
		'н': true,
		'п': true,
		'р': true,
		'с': true,
		'т': true,
		'ф': true,
		'х': true,
		'ц': true,
		'ч': true,
		'ш': true,
		'щ': true,
	}
)

func MetaphoneRu(rs []rune) []rune {
	res := make([]rune, 0, len(rs))
	i := 0
	l := len(rs)
	for i < l {
		r := rs[i]
		i++

		if i < l {
			switch r {
			case 'й', 'и':
				switch rs[i] {
				case 'о', 'е':
					res = append(res, 'и')
					i++
					continue
				}
			case 'т', 'д':
				if rs[i] == 'с' {
					res = append(res, 'ц')
					i++
					continue
				}
			}

			if r == rs[i] {
				if _, ok := mRu3[r]; ok {
					res = append(res, r)
					i++
					continue
				}
			}
		}

		switch r {
		case 'ь':
			continue
		case 'ю':
			res = append(res, 'у')
			continue
		case 'о', 'ы', 'я':
			res = append(res, 'а')
			continue
		case 'е', 'ё', 'э':
			res = append(res, 'и')
			continue
		}

		if i == l {
			res = append(res, makeDeafConsonantRu(r))
			continue
		}

		rr := rs[i]
		if _, ok := mRu2[rr]; ok {
			res = append(res, makeDeafConsonantRu(r))
		} else {
			res = append(res, r)
		}
	}

	return res
}

func makeDeafConsonantRu(r rune) rune {
	res, replaced := mRu1[r]
	if replaced {
		return res
	}

	return r
}

var (
	// voiced: muffled
	mUa1 = map[rune]rune{
		'б': 'п',
		'з': 'с',
		'д': 'т',
		'в': 'ф',
		'г': 'к',
	}

	// Consonants except Л, М, Н, Р
	mUa2 = map[rune]bool{
		'б': true,
		'в': true,
		'г': true,
		'д': true,
		'ж': true,
		'з': true,
		'й': true,
		'к': true,
		'п': true,
		'с': true,
		'т': true,
		'ф': true,
		'х': true,
		'ц': true,
		'ч': true,
		'ш': true,
		'щ': true,
	}

	// All consonants
	mUa3 = map[rune]bool{
		'б': true,
		'в': true,
		'г': true,
		'д': true,
		'ж': true,
		'з': true,
		'й': true,
		'к': true,
		'л': true,
		'м': true,
		'н': true,
		'п': true,
		'р': true,
		'с': true,
		'т': true,
		'ф': true,
		'х': true,
		'ц': true,
		'ч': true,
		'ш': true,
		'щ': true,
	}
)

func MetaphoneUa(rs []rune) []rune {
	res := make([]rune, 0, len(rs))
	i := 0
	l := len(rs)
	for i < l {
		r := rs[i]
		i++

		if i < l {
			switch r {
			case 'й', 'и':
				switch rs[i] {
				case 'о', 'е':
					res = append(res, 'и')
					i++
					continue
				}
			case 'т', 'д':
				if rs[i] == 'с' {
					res = append(res, 'ц')
					i++
					continue
				}
			}

			if r == rs[i] {
				if _, ok := mUa3[r]; ok {
					res = append(res, r)
					i++
					continue
				}
			}
		}

		switch r {
		case 'ь':
			continue
		case 'ю':
			res = append(res, 'у')
			continue
		case 'ї':
			res = append(res, 'і')
			continue
		case 'о', 'я':
			res = append(res, 'а')
			continue
		case 'е', 'є':
			res = append(res, 'и')
			continue
		}

		if i == l {
			res = append(res, makeDeafConsonantUa(r))
			continue
		}

		rr := rs[i]
		if _, ok := mUa2[rr]; ok {
			res = append(res, makeDeafConsonantUa(r))
		} else {
			res = append(res, r)
		}
	}

	return res
}

func makeDeafConsonantUa(r rune) rune {
	res, replaced := mUa1[r]
	if replaced {
		return res
	}

	return r
}
