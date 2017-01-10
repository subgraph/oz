package parser

var _gosecco_tokenizer_actions []int8 = []int8{0, 1, 0, 1, 1, 1, 2, 1, 9, 1, 10, 1, 11, 1, 12, 1, 13, 1, 14, 1, 15, 1, 16, 1, 17, 1, 18, 1, 19, 1, 20, 1, 21, 1, 22, 1, 23, 1, 24, 1, 25, 1, 26, 1, 27, 1, 28, 1, 29, 1, 30, 1, 31, 1, 32, 1, 33, 1, 34, 1, 35, 1, 36, 1, 37, 1, 38, 1, 39, 1, 40, 1, 41, 1, 42, 1, 43, 1, 44, 2, 2, 3, 2, 2, 4, 2, 2, 5, 2, 2, 6, 2, 2, 7, 2, 2, 8, 0}
var _gosecco_tokenizer_key_offsets []int16 = []int16{0, 2, 8, 45, 47, 48, 50, 56, 58, 60, 66, 68, 70, 71, 73, 80, 89, 98, 107, 116, 125, 134, 143, 152, 161, 170, 179, 188, 196, 204, 215, 224, 0}
var _gosecco_tokenizer_trans_keys []byte = []byte{48, 49, 48, 57, 65, 70, 97, 102, 9, 32, 33, 37, 38, 40, 41, 42, 43, 44, 45, 47, 48, 60, 61, 62, 70, 73, 78, 84, 91, 93, 94, 95, 97, 102, 105, 110, 116, 124, 126, 49, 57, 65, 90, 98, 122, 9, 32, 61, 38, 63, 66, 88, 98, 120, 48, 55, 48, 55, 48, 49, 48, 57, 65, 70, 97, 102, 48, 57, 60, 61, 61, 61, 62, 95, 48, 57, 65, 90, 97, 122, 65, 95, 97, 48, 57, 66, 90, 98, 122, 76, 95, 108, 48, 57, 65, 90, 97, 122, 83, 95, 115, 48, 57, 65, 90, 97, 122, 69, 95, 101, 48, 57, 65, 90, 97, 122, 78, 95, 110, 48, 57, 65, 90, 97, 122, 79, 95, 111, 48, 57, 65, 90, 97, 122, 84, 95, 116, 48, 57, 65, 90, 97, 122, 73, 95, 105, 48, 57, 65, 90, 97, 122, 78, 95, 110, 48, 57, 65, 90, 97, 122, 82, 95, 114, 48, 57, 65, 90, 97, 122, 85, 95, 117, 48, 57, 65, 90, 97, 122, 69, 95, 101, 48, 57, 65, 90, 97, 122, 95, 114, 48, 57, 65, 90, 97, 122, 95, 103, 48, 57, 65, 90, 97, 122, 72, 76, 95, 48, 53, 54, 57, 65, 90, 97, 122, 95, 48, 53, 54, 57, 65, 90, 97, 122, 124, 0}
var _gosecco_tokenizer_single_lengths []int8 = []int8{0, 0, 31, 2, 1, 2, 4, 0, 0, 0, 0, 2, 1, 2, 1, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 2, 2, 3, 1, 1, 0}
var _gosecco_tokenizer_range_lengths []int8 = []int8{1, 3, 3, 0, 0, 0, 1, 1, 1, 3, 1, 0, 0, 0, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 4, 4, 0, 0}
var _gosecco_tokenizer_index_offsets []int16 = []int16{0, 2, 6, 41, 44, 46, 49, 55, 57, 59, 63, 65, 68, 70, 73, 78, 85, 92, 99, 106, 113, 120, 127, 134, 141, 148, 155, 162, 168, 174, 182, 188, 0}
var _gosecco_tokenizer_trans_cond_spaces []int8 = []int8{-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, 0}
var _gosecco_tokenizer_trans_offsets []int16 = []int16{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62, 63, 64, 65, 66, 67, 68, 69, 70, 71, 72, 73, 74, 75, 76, 77, 78, 79, 80, 81, 82, 83, 84, 85, 86, 87, 88, 89, 90, 91, 92, 93, 94, 95, 96, 97, 98, 99, 100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110, 111, 112, 113, 114, 115, 116, 117, 118, 119, 120, 121, 122, 123, 124, 125, 126, 127, 128, 129, 130, 131, 132, 133, 134, 135, 136, 137, 138, 139, 140, 141, 142, 143, 144, 145, 146, 147, 148, 149, 150, 151, 152, 153, 154, 155, 156, 157, 158, 159, 160, 161, 162, 163, 164, 165, 166, 167, 168, 169, 170, 171, 172, 173, 174, 175, 176, 177, 178, 179, 180, 181, 182, 183, 184, 185, 186, 187, 188, 189, 190, 191, 192, 193, 194, 195, 196, 197, 198, 199, 200, 201, 202, 203, 204, 205, 206, 207, 208, 209, 210, 211, 212, 213, 214, 215, 216, 217, 218, 219, 220, 0}
var _gosecco_tokenizer_trans_lengths []int8 = []int8{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0}
var _gosecco_tokenizer_cond_keys []int8 = []int8{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
var _gosecco_tokenizer_cond_targs []int8 = []int8{8, 2, 9, 9, 9, 2, 3, 3, 4, 2, 5, 2, 2, 2, 2, 2, 2, 2, 6, 11, 12, 13, 15, 19, 20, 24, 2, 2, 2, 14, 27, 15, 19, 20, 24, 31, 2, 10, 14, 14, 2, 3, 3, 2, 2, 2, 2, 2, 2, 0, 1, 0, 1, 7, 2, 7, 2, 8, 2, 9, 9, 9, 2, 10, 2, 2, 2, 2, 2, 2, 2, 2, 2, 14, 14, 14, 14, 2, 16, 14, 16, 14, 14, 14, 2, 17, 14, 17, 14, 14, 14, 2, 18, 14, 18, 14, 14, 14, 2, 14, 14, 14, 14, 14, 14, 2, 14, 14, 14, 14, 14, 14, 2, 21, 14, 21, 14, 14, 14, 2, 22, 14, 22, 14, 14, 14, 2, 23, 14, 23, 14, 14, 14, 2, 14, 14, 14, 14, 14, 14, 2, 25, 14, 25, 14, 14, 14, 2, 26, 14, 26, 14, 14, 14, 2, 14, 14, 14, 14, 14, 14, 2, 14, 28, 14, 14, 14, 2, 14, 29, 14, 14, 14, 2, 30, 30, 14, 14, 14, 14, 14, 2, 14, 14, 14, 14, 14, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 0}
var _gosecco_tokenizer_cond_actions []int8 = []int8{0, 75, 0, 0, 0, 75, 0, 0, 0, 15, 0, 39, 43, 11, 7, 47, 9, 13, 5, 0, 0, 0, 0, 0, 0, 0, 41, 45, 23, 94, 0, 0, 0, 0, 0, 0, 29, 0, 94, 94, 49, 0, 0, 71, 37, 69, 19, 17, 61, 0, 0, 0, 0, 0, 59, 0, 55, 0, 57, 0, 0, 0, 53, 0, 59, 25, 33, 65, 31, 73, 35, 27, 67, 94, 94, 94, 94, 77, 0, 94, 0, 94, 94, 94, 51, 0, 94, 0, 94, 94, 94, 51, 0, 94, 0, 94, 94, 94, 51, 91, 94, 91, 94, 94, 94, 51, 82, 94, 82, 94, 94, 94, 51, 0, 94, 0, 94, 94, 94, 51, 0, 94, 0, 94, 94, 94, 51, 0, 94, 0, 94, 94, 94, 51, 85, 94, 85, 94, 94, 94, 51, 0, 94, 0, 94, 94, 94, 51, 0, 94, 0, 94, 94, 94, 51, 88, 94, 88, 94, 94, 94, 51, 94, 0, 94, 94, 94, 51, 94, 0, 94, 94, 94, 51, 0, 0, 94, 79, 94, 94, 94, 51, 94, 79, 94, 94, 94, 51, 21, 63, 75, 75, 71, 69, 61, 59, 55, 57, 53, 59, 65, 73, 67, 77, 51, 51, 51, 51, 51, 51, 51, 51, 51, 51, 51, 51, 51, 51, 51, 51, 63, 0}
var _gosecco_tokenizer_to_state_actions []int8 = []int8{0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
var _gosecco_tokenizer_from_state_actions []int8 = []int8{0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
var _gosecco_tokenizer_eof_trans_indexed []int8 = []int8{3, 3, 0, 17, 18, 20, 23, 24, 25, 26, 23, 27, 30, 32, 35, 36, 36, 36, 36, 36, 36, 36, 36, 36, 36, 36, 36, 36, 36, 36, 36, 37, 0}
var _gosecco_tokenizer_eof_trans_direct []int16 = []int16{191, 192, 0, 193, 194, 195, 196, 197, 198, 199, 200, 201, 202, 203, 204, 205, 206, 207, 208, 209, 210, 211, 212, 213, 214, 215, 216, 217, 218, 219, 220, 221, 0}
var _gosecco_tokenizer_nfa_targs []int8 = []int8{0, 0}
var _gosecco_tokenizer_nfa_offsets []int8 = []int8{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
var _gosecco_tokenizer_nfa_push_actions []int8 = []int8{0, 0}
var _gosecco_tokenizer_nfa_pop_trans []int8 = []int8{0, 0}
var gosecco_tokenizer_start int = 2
var gosecco_tokenizer_first_final int = 2
var gosecco_tokenizer_error int = -1
var gosecco_tokenizer_en_main int = 2

func tokenizeRaw(data []byte, f func(token, []byte), tokenError func(int, int, []byte) error) error {
	var cs, act int
	p, pe := 0, len(data)
	ts, te := 0, 0
	eof := pe

	{
		cs = int(gosecco_tokenizer_start)
		ts = 0
		te = 0
		act = 0
	}

	{
		var _klen int
		var _trans uint = 0
		var _cond uint = 0
		var _acts int
		var _nacts uint
		var _keys int
		var _ckeys int
		var _cpc int
		if p == pe {
			goto _test_eof

		}
	_resume:
		_acts = int(_gosecco_tokenizer_from_state_actions[cs])
		_nacts = uint(_gosecco_tokenizer_actions[_acts])
		_acts += 1
		for _nacts > 0 {
			{
				switch _gosecco_tokenizer_actions[_acts] {
				case 1:
					{
						{
							ts = p
						}
					}

					break

				}
				_nacts -= 1
				_acts += 1
			}

		}
		_keys = int(_gosecco_tokenizer_key_offsets[cs])
		_trans = uint(_gosecco_tokenizer_index_offsets[cs])
		_klen = int(_gosecco_tokenizer_single_lengths[cs])
		if _klen > 0 {
			{
				var _lower int
				var _mid int
				var _upper int
				_lower = _keys
				_upper = _keys + _klen - 1
				for {
					{
						if _upper < _lower {
							break

						}
						_mid = _lower + ((_upper - _lower) >> 1)
						switch {
						case (data[p]) < _gosecco_tokenizer_trans_keys[_mid]:
							_upper = _mid - 1

						case (data[p]) > _gosecco_tokenizer_trans_keys[_mid]:
							_lower = _mid + 1

						default:
							{
								_trans += uint((_mid - _keys))
								goto _match
							}

						}
					}

				}
				_keys += _klen
				_trans += uint(_klen)
			}

		}
		_klen = int(_gosecco_tokenizer_range_lengths[cs])
		if _klen > 0 {
			{
				var _lower int
				var _mid int
				var _upper int
				_lower = _keys
				_upper = _keys + (_klen << 1) - 2
				for {
					{
						if _upper < _lower {
							break

						}
						_mid = _lower + (((_upper - _lower) >> 1) & ^1)
						switch {
						case (data[p]) < _gosecco_tokenizer_trans_keys[_mid]:
							_upper = _mid - 2

						case (data[p]) > _gosecco_tokenizer_trans_keys[_mid+1]:
							_lower = _mid + 2

						default:
							{
								_trans += uint(((_mid - _keys) >> 1))
								goto _match
							}

						}
					}

				}
				_trans += uint(_klen)
			}

		}

	_match:
		_ckeys = int(_gosecco_tokenizer_trans_offsets[_trans])
		_klen = int(_gosecco_tokenizer_trans_lengths[_trans])
		_cond = uint(_gosecco_tokenizer_trans_offsets[_trans])
		_cpc = 0
		{
			var _lower int
			var _mid int
			var _upper int
			_lower = _ckeys
			_upper = _ckeys + _klen - 1
			for {
				{
					if _upper < _lower {
						break

					}
					_mid = _lower + ((_upper - _lower) >> 1)
					switch {
					case _cpc < int(_gosecco_tokenizer_cond_keys[_mid]):
						_upper = _mid - 1

					case _cpc > int(_gosecco_tokenizer_cond_keys[_mid]):
						_lower = _mid + 1

					default:
						{
							_cond += uint((_mid - _ckeys))
							goto _match_cond
						}

					}
				}

			}
			cs = -1
			goto _again
		}

	_match_cond:
		cs = int(_gosecco_tokenizer_cond_targs[_cond])
		if _gosecco_tokenizer_cond_actions[_cond] == 0 {
			goto _again

		}
		_acts = int(_gosecco_tokenizer_cond_actions[_cond])
		_nacts = uint(_gosecco_tokenizer_actions[_acts])
		_acts += 1
		for _nacts > 0 {
			{
				switch _gosecco_tokenizer_actions[_acts] {
				case 2:
					{
						{
							te = p + 1
						}
					}

					break
				case 3:
					{
						{
							act = 1
						}
					}

					break
				case 4:
					{
						{
							act = 2
						}
					}

					break
				case 5:
					{
						{
							act = 3
						}
					}

					break
				case 6:
					{
						{
							act = 4
						}
					}

					break
				case 7:
					{
						{
							act = 5
						}
					}

					break
				case 8:
					{
						{
							act = 6
						}
					}

					break
				case 9:
					{
						{
							te = p + 1
							{
								f(ADD, nil)
							}
						}
					}

					break
				case 10:
					{
						{
							te = p + 1
							{
								f(SUB, nil)
							}
						}
					}

					break
				case 11:
					{
						{
							te = p + 1
							{
								f(MUL, nil)
							}
						}
					}

					break
				case 12:
					{
						{
							te = p + 1
							{
								f(DIV, nil)
							}
						}
					}

					break
				case 13:
					{
						{
							te = p + 1
							{
								f(MOD, nil)
							}
						}
					}

					break
				case 14:
					{
						{
							te = p + 1
							{
								f(BITSET, nil)
							}
						}
					}

					break
				case 15:
					{
						{
							te = p + 1
							{
								f(LAND, nil)
							}
						}
					}

					break
				case 16:
					{
						{
							te = p + 1
							{
								f(LOR, nil)
							}
						}
					}

					break
				case 17:
					{
						{
							te = p + 1
							{
								f(XOR, nil)
							}
						}
					}

					break
				case 18:
					{
						{
							te = p + 1
							{
								f(LSH, nil)
							}
						}
					}

					break
				case 19:
					{
						{
							te = p + 1
							{
								f(RSH, nil)
							}
						}
					}

					break
				case 20:
					{
						{
							te = p + 1
							{
								f(INV, nil)
							}
						}
					}

					break
				case 21:
					{
						{
							te = p + 1
							{
								f(EQL, nil)
							}
						}
					}

					break
				case 22:
					{
						{
							te = p + 1
							{
								f(LTE, nil)
							}
						}
					}

					break
				case 23:
					{
						{
							te = p + 1
							{
								f(GTE, nil)
							}
						}
					}

					break
				case 24:
					{
						{
							te = p + 1
							{
								f(NEQ, nil)
							}
						}
					}

					break
				case 25:
					{
						{
							te = p + 1
							{
								f(LPAREN, nil)
							}
						}
					}

					break
				case 26:
					{
						{
							te = p + 1
							{
								f(LBRACK, nil)
							}
						}
					}

					break
				case 27:
					{
						{
							te = p + 1
							{
								f(RPAREN, nil)
							}
						}
					}

					break
				case 28:
					{
						{
							te = p + 1
							{
								f(RBRACK, nil)
							}
						}
					}

					break
				case 29:
					{
						{
							te = p + 1
							{
								f(COMMA, nil)
							}
						}
					}

					break
				case 30:
					{
						{
							te = p + 1
							{
								return tokenError(ts, te, data)
							}
						}
					}

					break
				case 31:
					{
						{
							te = p
							p = p - 1
							{
								f(IDENT, data[ts:te])
							}
						}
					}

					break
				case 32:
					{
						{
							te = p
							p = p - 1
							{
								f(INT, data[ts:te])
							}
						}
					}

					break
				case 33:
					{
						{
							te = p
							p = p - 1
							{
								f(INT, data[ts:te])
							}
						}
					}

					break
				case 34:
					{
						{
							te = p
							p = p - 1
							{
								f(INT, data[ts:te])
							}
						}
					}

					break
				case 35:
					{
						{
							te = p
							p = p - 1
							{
								f(INT, data[ts:te])
							}
						}
					}

					break
				case 36:
					{
						{
							te = p
							p = p - 1
							{
								f(AND, nil)
							}
						}
					}

					break
				case 37:
					{
						{
							te = p
							p = p - 1
							{
								f(OR, nil)
							}
						}
					}

					break
				case 38:
					{
						{
							te = p
							p = p - 1
							{
								f(LT, nil)
							}
						}
					}

					break
				case 39:
					{
						{
							te = p
							p = p - 1
							{
								f(GT, nil)
							}
						}
					}

					break
				case 40:
					{
						{
							te = p
							p = p - 1
							{
								f(NOT, nil)
							}
						}
					}

					break
				case 41:
					{
						{
							te = p
							p = p - 1
						}
					}

					break
				case 42:
					{
						{
							te = p
							p = p - 1
							{
								return tokenError(ts, te, data)
							}
						}
					}

					break
				case 43:
					{
						{
							p = (te) - 1
							{
								f(INT, data[ts:te])
							}
						}
					}

					break
				case 44:
					{
						{
							switch act {
							case 1:
								p = (te) - 1
								{
									f(ARG, data[ts:te])
								}

								break
							case 2:
								p = (te) - 1
								{
									f(IN, nil)
								}

								break
							case 3:
								p = (te) - 1
								{
									f(NOTIN, nil)
								}

								break
							case 4:
								p = (te) - 1
								{
									f(TRUE, nil)
								}

								break
							case 5:
								p = (te) - 1
								{
									f(FALSE, nil)
								}

								break
							case 6:
								p = (te) - 1
								{
									f(IDENT, data[ts:te])
								}

								break

							}
						}
					}

					break

				}
				_nacts -= 1
				_acts += 1
			}

		}

	_again:
		_acts = int(_gosecco_tokenizer_to_state_actions[cs])
		_nacts = uint(_gosecco_tokenizer_actions[_acts])
		_acts += 1
		for _nacts > 0 {
			{
				switch _gosecco_tokenizer_actions[_acts] {
				case 0:
					{
						{
							ts = 0
						}
					}

					break

				}
				_nacts -= 1
				_acts += 1
			}

		}
		p += 1
		if p != pe {
			goto _resume

		}

	_test_eof:
		{
		}
		if p == eof {
			{
				if _gosecco_tokenizer_eof_trans_direct[cs] > 0 {
					{
						_trans = uint(_gosecco_tokenizer_eof_trans_direct[cs]) - 1
						_cond = uint(_gosecco_tokenizer_trans_offsets[_trans])
						goto _match_cond
					}

				}
			}

		}

	}
	return nil
}
